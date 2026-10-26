package repair

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"comics/internal/identity"
)

const PlanSchemaVersion = 1

type Decision string

const (
	DecisionReview      Decision = "review"
	DecisionRestoreKeys Decision = "restore_keys"
	DecisionMerge       Decision = "merge"
	DecisionBlock       Decision = "block"
)

type Review struct {
	Approved    bool     `json:"approved"`
	Decision    Decision `json:"decision"`
	Reviewer    string   `json:"reviewer"`
	ReviewedAt  string   `json:"reviewed_at"`
	CanonicalID int      `json:"canonical_id,omitempty"`
	MergeIDs    []int    `json:"merge_ids,omitempty"`
	Notes       string   `json:"notes"`
}

type Signals struct {
	ExactKey      bool    `json:"exact_key"`
	ExactTitle    bool    `json:"exact_title"`
	TitlePrefix   bool    `json:"title_prefix"`
	TitleTokens   bool    `json:"title_tokens"`
	SameType      bool    `json:"same_type"`
	NovelBoundary bool    `json:"novel_boundary"`
	ChaptersClose bool    `json:"chapters_close"`
	ChaptersFar   bool    `json:"chapters_far"`
	ChapterSpread int     `json:"chapter_spread"`
	ChapterRatio  float64 `json:"chapter_ratio"`
	GroupSize     int     `json:"group_size"`
}

type Group struct {
	GroupID              string                    `json:"group_id"`
	Reason               string                    `json:"reason"`
	Risk                 string                    `json:"risk"`
	Keyless              ComicSnapshot             `json:"keyless"`
	Owners               []ComicSnapshot           `json:"owners"`
	ProposedKeys         []identity.ScopedTitleKey `json:"proposed_keys"`
	SuggestedCanonicalID int                       `json:"suggested_canonical_id,omitempty"`
	Signals              Signals                   `json:"signals"`
	Review               Review                    `json:"review"`
}

type Plan struct {
	SchemaVersion       int          `json:"schema_version"`
	GeneratedAt         string       `json:"generated_at"`
	DatabaseFingerprint string       `json:"database_fingerprint"`
	Summary             AuditSummary `json:"summary"`
	Groups              []Group      `json:"groups"`
}

const (
	maxLowRiskGroupSize = 4
	chapterSpreadClose  = 15
	chapterRatioClose   = 0.85
	chapterSpreadFar    = 50
	chapterRatioFar     = 0.50
)

func BuildPlan(audit Audit, generatedAt time.Time) Plan {
	groups := make([]Group, 0, len(audit.Records))
	for _, record := range audit.Records {
		groups = append(groups, buildGroup(record))
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].GroupID < groups[j].GroupID
	})
	return Plan{
		SchemaVersion:       PlanSchemaVersion,
		GeneratedAt:         generatedAt.UTC().Format(time.RFC3339),
		DatabaseFingerprint: audit.DatabaseFingerprint,
		Summary:             audit.Summary,
		Groups:              groups,
	}
}

func buildGroup(record KeylessRecord) Group {
	owners := ownerSnapshots(record.Owners)
	members := append([]ComicSnapshot{record.Comic}, owners...)
	signals := ComputeSignals(members, record.ProposedKeys)
	risk := classifyRisk(record, members, signals)
	suggested := 0
	if !signals.NovelBoundary {
		suggested = lowestID(members)
	}
	return Group{
		GroupID:              fmt.Sprintf("keyless:%d", record.Comic.ID),
		Reason:               record.Classification,
		Risk:                 risk,
		Keyless:              record.Comic,
		Owners:               owners,
		ProposedKeys:         append([]identity.ScopedTitleKey(nil), record.ProposedKeys...),
		SuggestedCanonicalID: suggested,
		Signals:              signals,
		Review: Review{
			Approved:   false,
			Decision:   DecisionReview,
			Reviewer:   "",
			ReviewedAt: "",
			Notes:      "",
		},
	}
}

func ownerSnapshots(owners []KeyOwner) []ComicSnapshot {
	seen := map[int]struct{}{}
	out := make([]ComicSnapshot, 0, len(owners))
	for _, owner := range owners {
		if _, ok := seen[owner.Comic.ID]; ok {
			continue
		}
		seen[owner.Comic.ID] = struct{}{}
		out = append(out, owner.Comic)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func classifyRisk(record KeylessRecord, members []ComicSnapshot, signals Signals) string {
	if record.Classification == ClassificationBlockedNoKey || signals.NovelBoundary {
		return "blocked"
	}
	if len(members) > maxLowRiskGroupSize || signals.ChaptersFar || !signals.SameType ||
		(!signals.ExactKey && !signals.ExactTitle && !signals.TitlePrefix && !signals.TitleTokens) {
		return "review"
	}
	if signals.ExactKey && signals.SameType && signals.ChaptersClose && len(members) <= maxLowRiskGroupSize {
		return "low"
	}
	return "review"
}

func lowestID(members []ComicSnapshot) int {
	if len(members) == 0 {
		return 0
	}
	id := members[0].ID
	for _, member := range members[1:] {
		if member.ID < id {
			id = member.ID
		}
	}
	return id
}

func ComputeSignals(members []ComicSnapshot, proposed []identity.ScopedTitleKey) Signals {
	signals := Signals{GroupSize: len(members)}
	if len(members) == 0 {
		return signals
	}

	minChap, maxChap := members[0].CurrentChap, members[0].CurrentChap
	comType := members[0].ComType
	signals.SameType = true
	for _, member := range members {
		if member.CurrentChap < minChap {
			minChap = member.CurrentChap
		}
		if member.CurrentChap > maxChap {
			maxChap = member.CurrentChap
		}
		if member.ComType != comType {
			signals.SameType = false
		}
		for _, title := range member.Titles {
			if identity.IsNovelIdentity(title, member.ComType) {
				signals.NovelBoundary = true
			}
		}
	}
	spread := maxChap - minChap
	ratio := 1.0
	if maxChap > 0 {
		ratio = float64(minChap) / float64(maxChap)
	}
	signals.ChapterSpread = spread
	signals.ChapterRatio = roundRatio(ratio)
	signals.ChaptersClose = spread <= chapterSpreadClose ||
		(maxChap >= 20 && ratio >= chapterRatioClose)
	signals.ChaptersFar = spread > chapterSpreadFar ||
		(maxChap >= 30 && ratio < chapterRatioFar)

	primary := primaryTitle(members[0])
	for _, member := range members[1:] {
		stored := primaryTitle(member)
		if identity.TitleMatchKey(primary) == identity.TitleMatchKey(stored) && identity.TitleMatchKey(primary) != "" {
			signals.ExactKey = true
		}
		if equalFold(primary, stored) {
			signals.ExactTitle = true
		}
		if identity.TitlesArePrefixMatch(primary, stored) || identity.TitlesArePrefixMatch(stored, primary) {
			signals.TitlePrefix = true
		}
		if identity.TitlesMatchBySignificantTokens(primary, stored) {
			signals.TitleTokens = true
		}
	}
	if len(proposed) == 0 {
		return signals
	}
	for _, key := range proposed {
		if key.Scope == "novel" {
			signals.NovelBoundary = true
		}
	}
	return signals
}

func roundRatio(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func primaryTitle(snapshot ComicSnapshot) string {
	if len(snapshot.Titles) == 0 {
		return ""
	}
	return snapshot.Titles[0]
}

func equalFold(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func ValidatePlan(plan Plan, audit Audit) error {
	if plan.SchemaVersion != PlanSchemaVersion {
		return fmt.Errorf("unsupported plan schema %d", plan.SchemaVersion)
	}
	if plan.DatabaseFingerprint != audit.DatabaseFingerprint {
		return fmt.Errorf("plan fingerprint does not match current database")
	}
	seen := map[string]struct{}{}
	auditByGroup := map[string]KeylessRecord{}
	for _, record := range audit.Records {
		auditByGroup[fmt.Sprintf("keyless:%d", record.Comic.ID)] = record
	}
	for _, group := range plan.Groups {
		if _, exists := seen[group.GroupID]; exists {
			return fmt.Errorf("duplicate group id %s", group.GroupID)
		}
		seen[group.GroupID] = struct{}{}
		record, ok := auditByGroup[group.GroupID]
		if !ok {
			return fmt.Errorf("group %s not present in current audit", group.GroupID)
		}
		if err := validateGroupSnapshot(group, record); err != nil {
			return err
		}
		if err := validateGroupReview(group, record); err != nil {
			return err
		}
	}
	return nil
}

func validateGroupSnapshot(group Group, record KeylessRecord) error {
	if group.Keyless.ID != record.Comic.ID {
		return fmt.Errorf("group %s keyless id mismatch", group.GroupID)
	}
	if !equalSnapshots(group.Keyless, record.Comic) {
		return fmt.Errorf("group %s keyless snapshot changed", group.GroupID)
	}
	if !equalScopedKeys(group.ProposedKeys, record.ProposedKeys) {
		return fmt.Errorf("group %s proposed keys changed", group.GroupID)
	}
	return nil
}

func validateGroupReview(group Group, record KeylessRecord) error {
	review := group.Review
	if !review.Approved {
		if review.Decision == DecisionRestoreKeys || review.Decision == DecisionMerge {
			return fmt.Errorf("group %s has mutation decision without approval", group.GroupID)
		}
		return nil
	}
	if review.Reviewer == "" || review.ReviewedAt == "" {
		return fmt.Errorf("group %s missing reviewer metadata", group.GroupID)
	}
	if _, err := time.Parse(time.RFC3339, review.ReviewedAt); err != nil {
		return fmt.Errorf("group %s invalid reviewed_at: %w", group.GroupID, err)
	}
	switch review.Decision {
	case DecisionReview, DecisionBlock:
		return fmt.Errorf("group %s cannot apply decision %q", group.GroupID, review.Decision)
	case DecisionRestoreKeys:
		for _, owner := range record.Owners {
			if !owner.Comic.Deleted {
				return fmt.Errorf("group %s restore_keys blocked by active owner %d", group.GroupID, owner.Comic.ID)
			}
		}
	case DecisionMerge:
		if err := validateMergeReview(group, record); err != nil {
			return err
		}
	default:
		return fmt.Errorf("group %s unknown decision %q", group.GroupID, review.Decision)
	}
	return nil
}

func validateMergeReview(group Group, record KeylessRecord) error {
	if len(record.Owners) == 0 {
		return fmt.Errorf("group %s merge has no owners", group.GroupID)
	}
	members := append([]ComicSnapshot{record.Comic}, ownerSnapshots(record.Owners)...)
	if ComputeSignals(members, record.ProposedKeys).NovelBoundary {
		return fmt.Errorf("group %s merge crosses novel boundary", group.GroupID)
	}
	review := group.Review
	if review.CanonicalID == 0 {
		return fmt.Errorf("group %s merge missing canonical id", group.GroupID)
	}
	memberIDs := map[int]struct{}{}
	for _, member := range members {
		memberIDs[member.ID] = struct{}{}
	}
	if _, ok := memberIDs[review.CanonicalID]; !ok {
		return fmt.Errorf("group %s canonical %d not in group", group.GroupID, review.CanonicalID)
	}
	seenMerge := map[int]struct{}{}
	for _, mergeID := range review.MergeIDs {
		if mergeID == review.CanonicalID {
			return fmt.Errorf("group %s canonical in merge ids", group.GroupID)
		}
		if _, ok := memberIDs[mergeID]; !ok {
			return fmt.Errorf("group %s merge id %d not in group", group.GroupID, mergeID)
		}
		if _, dup := seenMerge[mergeID]; dup {
			return fmt.Errorf("group %s duplicate merge id %d", group.GroupID, mergeID)
		}
		seenMerge[mergeID] = struct{}{}
	}
	for id := range memberIDs {
		if id == review.CanonicalID {
			continue
		}
		if _, ok := seenMerge[id]; !ok {
			return fmt.Errorf("group %s omits group member %d from merge ids", group.GroupID, id)
		}
	}
	return nil
}

func FilterApproved(plan Plan, groupID string, allApproved bool) ([]Group, error) {
	if groupID != "" && allApproved {
		return nil, fmt.Errorf("use only one of group-id or all-approved")
	}
	if groupID == "" && !allApproved {
		return nil, fmt.Errorf("group-id or all-approved is required")
	}
	if groupID != "" {
		for _, group := range plan.Groups {
			if group.GroupID == groupID {
				if !group.Review.Approved {
					return nil, fmt.Errorf("group %s is not approved", groupID)
				}
				return []Group{group}, nil
			}
		}
		return nil, fmt.Errorf("group %s not found", groupID)
	}
	out := make([]Group, 0)
	for _, group := range plan.Groups {
		if group.Review.Approved {
			out = append(out, group)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no approved groups in plan")
	}
	return out, nil
}

func equalSnapshots(a, b ComicSnapshot) bool {
	if a.ID != b.ID || a.CurrentChap != b.CurrentChap || a.ComType != b.ComType ||
		a.Track != b.Track || a.Deleted != b.Deleted || len(a.Titles) != len(b.Titles) {
		return false
	}
	for i := range a.Titles {
		if a.Titles[i] != b.Titles[i] {
			return false
		}
	}
	return true
}

func equalScopedKeys(a, b []identity.ScopedTitleKey) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
