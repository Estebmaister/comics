package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"comics/internal/repo/comics"
)

type mergePlan struct {
	GeneratedAt string       `json:"generated_at"`
	GroupCount  int          `json:"group_count"`
	Groups      []mergeGroup `json:"groups"`
}

type mergeGroup struct {
	GroupID     string `json:"group_id"`
	Reason      string `json:"reason"`
	CanonicalID int    `json:"canonical_id"`
	MergeIDs    []int  `json:"merge_ids"`
	IDs         []int  `json:"ids"`
	Scope       string `json:"scope"`
	TitleKey    string `json:"title_key"`
	Risk        string `json:"risk"`
	Approved    bool   `json:"approved"`
}

func main() {
	dbPath := flag.String("db", "", "sqlite database path (default: COMICS_SQLITE_PATH or db/comics.db)")
	integrityAudit := flag.Bool("audit", false, "run read-only discovery integrity audit")
	exportPlan := flag.String("export-plan", "", "write discovery key collision merge plan JSON to path")
	flag.Parse()

	ctx := context.Background()
	repo, err := comics.NewSQLiteComicRepositoryReadOnly(*dbPath)
	if err != nil {
		fatal(err)
	}
	defer repo.Close()

	if *integrityAudit {
		audit, err := repo.AuditDiscoveryIntegrity(ctx)
		if err != nil {
			fatal(err)
		}
		s := audit.Summary
		fmt.Printf(
			"discovery audit: total=%d active=%d keys=%d active_owners=%d keyless=%d conflicts=%d restorable=%d blocked=%d deleted_keys=%d orphan_keys=%d fingerprint=%s\n",
			s.TotalComics,
			s.ActiveComics,
			s.KeyRows,
			s.ActiveKeyOwners,
			s.KeylessActive,
			s.ActiveConflicts,
			s.Restorable,
			s.BlockedNoKey,
			s.DeletedKeyRows,
			s.OrphanKeyRows,
			audit.DatabaseFingerprint,
		)
		return
	}

	collisions, err := repo.ListDiscoveryKeyCollisions(ctx)
	if err != nil {
		fatal(err)
	}

	plan := buildPlan(collisions)
	if *exportPlan == "" {
		fmt.Printf("discovery collisions: %d groups\n", plan.GroupCount)
		for _, group := range plan.Groups {
			fmt.Printf("- %s %s:%s ids=%v canonical=%d merge=%v risk=%s\n",
				group.Reason, group.Scope, group.TitleKey, group.IDs, group.CanonicalID, group.MergeIDs, group.Risk)
		}
		return
	}

	payload, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*exportPlan, payload, 0o600); err != nil {
		fatal(err)
	}
	fmt.Printf("wrote %s\n", *exportPlan)
}

func buildPlan(collisions []comics.DiscoveryCollision) mergePlan {
	groups := make([]mergeGroup, 0, len(collisions))
	for _, collision := range collisions {
		ids := append([]int(nil), collision.ComicIDs...)
		if len(ids) < 2 {
			continue
		}
		canonical := ids[0]
		for _, id := range ids[1:] {
			if id < canonical {
				canonical = id
			}
		}
		mergeIDs := make([]int, 0, len(ids)-1)
		for _, id := range ids {
			if id == canonical {
				continue
			}
			mergeIDs = append(mergeIDs, id)
		}
		groups = append(groups, mergeGroup{
			GroupID:     fmt.Sprintf("discovery:%s:%s", collision.Scope, collision.TitleKey),
			Reason:      "discovery_key_collision",
			CanonicalID: canonical,
			MergeIDs:    mergeIDs,
			IDs:         ids,
			Scope:       collision.Scope,
			TitleKey:    collision.TitleKey,
			Risk:        "review",
			Approved:    false,
		})
	}
	return mergePlan{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		GroupCount:  len(groups),
		Groups:      groups,
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
