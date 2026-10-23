package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"comics/internal/repo/comics"
	"comics/internal/usecase"
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
	audit := flag.Bool("audit", false, "print discovery key collision summary")
	exportPlan := flag.String("export-plan", "", "write merge plan JSON to path")
	importPlan := flag.String("import-plan", "", "read merge plan JSON from path")
	apply := flag.Bool("apply", false, "apply approved groups from import plan")
	flag.Parse()

	ctx := context.Background()
	repo, err := comics.NewSQLiteComicRepository("")
	if err != nil {
		fatal(err)
	}
	defer repo.Close()

	collisions, err := repo.ListDiscoveryKeyCollisions(ctx)
	if err != nil {
		fatal(err)
	}

	if *importPlan == "" {
		plan := buildPlan(collisions)
		if *audit || *exportPlan == "" {
			fmt.Printf("discovery collisions: %d groups\n", plan.GroupCount)
			for _, group := range plan.Groups {
				fmt.Printf("- %s %s:%s ids=%v canonical=%d merge=%v risk=%s\n",
					group.Reason, group.Scope, group.TitleKey, group.IDs, group.CanonicalID, group.MergeIDs, group.Risk)
			}
		}
		if *exportPlan != "" {
			payload, err := json.MarshalIndent(plan, "", "  ")
			if err != nil {
				fatal(err)
			}
			if err := os.WriteFile(*exportPlan, payload, 0o600); err != nil {
				fatal(err)
			}
			fmt.Printf("wrote %s\n", *exportPlan)
		}
		return
	}
	data, err := os.ReadFile(*importPlan)
	if err != nil {
		fatal(err)
	}
	var plan mergePlan
	if err := json.Unmarshal(data, &plan); err != nil {
		fatal(err)
	}
	if !*apply {
		fmt.Printf("import plan loaded: %d groups (dry-run; pass --apply to merge)\n", len(plan.Groups))
		return
	}

	svc := usecase.NewComicService(repo, nil)
	applied := 0
	for _, group := range plan.Groups {
		if !group.Approved {
			continue
		}
		for _, mergeID := range group.MergeIDs {
			if _, err := svc.Merge(ctx, group.CanonicalID, mergeID); err != nil {
				fatal(fmt.Errorf("merge %d into %d: %w", mergeID, group.CanonicalID, err))
			}
			applied++
		}
	}
	fmt.Printf("applied %d merges\n", applied)
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
