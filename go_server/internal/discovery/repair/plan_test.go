package repair

import (
	"testing"
	"time"

	"comics/internal/identity"
)

func fixtureAudit() Audit {
	return Audit{
		DatabaseFingerprint: "fixture-fingerprint",
		Summary: AuditSummary{
			KeylessActive: 2,
		},
		Records: []KeylessRecord{
			{
				Comic: ComicSnapshot{
					ID: 5980, Titles: []string{"Hero"}, CurrentChap: 10, ComType: 3,
				},
				ProposedKeys: identityScoped("series", "hero"),
				Classification: ClassificationRestorable,
			},
			{
				Comic: ComicSnapshot{
					ID: 5981, Titles: []string{"The hero"}, CurrentChap: 8, ComType: 3,
				},
				Owners: []KeyOwner{{
					Key:   identityScoped("series", "hero")[0],
					Comic: ComicSnapshot{ID: 10, Titles: []string{"Hero"}, CurrentChap: 100, ComType: 3},
				}},
				ProposedKeys:   identityScoped("series", "hero"),
				Classification: ClassificationActiveConflict,
			},
		},
	}
}

func identityScoped(scope, key string) []identity.ScopedTitleKey {
	return []identity.ScopedTitleKey{{Scope: scope, TitleKey: key}}
}

func TestBuildPlanNeverApprovesGroups(t *testing.T) {
	plan := BuildPlan(fixtureAudit(), time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC))
	if len(plan.Groups) == 0 {
		t.Fatal("expected groups")
	}
	for _, group := range plan.Groups {
		if group.Review.Approved || group.Review.Decision != DecisionReview {
			t.Fatalf("export auto-approved group %#v", group)
		}
	}
}

func TestChapterSignals(t *testing.T) {
	signals := ComputeSignals([]ComicSnapshot{
		{ID: 10, Titles: []string{"Hero"}, CurrentChap: 100, ComType: 3},
		{ID: 11, Titles: []string{"The hero"}, CurrentChap: 90, ComType: 3},
	}, nil)
	if signals.ChapterSpread != 10 || signals.ChapterRatio != 0.9 ||
		!signals.ChaptersClose || signals.ChaptersFar || !signals.ExactKey ||
		!signals.SameType || signals.NovelBoundary {
		t.Fatalf("unexpected signals: %#v", signals)
	}
}

func TestValidatePlanRejectsUnreviewedAndStaleApply(t *testing.T) {
	plan := BuildPlan(fixtureAudit(), time.Now().UTC())
	plan.Groups[0].Review = Review{
		Approved:   true,
		Decision:   DecisionMerge,
		Reviewer:   "",
		ReviewedAt: "",
	}
	if err := ValidatePlan(plan, fixtureAudit()); err == nil {
		t.Fatal("expected missing reviewer to fail")
	}
	stale := fixtureAudit()
	stale.DatabaseFingerprint = "stale"
	if err := ValidatePlan(BuildPlan(fixtureAudit(), time.Now().UTC()), stale); err == nil {
		t.Fatal("expected fingerprint mismatch")
	}
}
