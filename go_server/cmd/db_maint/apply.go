package main

import (
	"context"
	"fmt"
	"os"

	"comics/domain"
	"comics/internal/discovery"
	"comics/internal/discovery/repair"
	"comics/internal/repo/comics"
	"comics/internal/usecase"
)

type discoveryApplyOptions struct {
	DBPath          string
	PlanPath        string
	BackupPath      string
	GroupID         string
	AllApproved     bool
	ConfirmReviewed bool
}

func applyDiscoveryRepair(ctx context.Context, opts discoveryApplyOptions) (repair.AuditSummary, repair.AuditSummary, error) {
	if !opts.ConfirmReviewed {
		return repair.AuditSummary{}, repair.AuditSummary{}, fmt.Errorf("--confirm-reviewed is required")
	}
	if opts.DBPath == "" {
		opts.DBPath = firstNonEmpty(os.Getenv("COMICS_SQLITE_PATH"), "../db/comics.db")
	}

	plan, err := repair.LoadPlanFile(opts.PlanPath)
	if err != nil {
		return repair.AuditSummary{}, repair.AuditSummary{}, err
	}
	groups, err := repair.FilterApproved(plan, opts.GroupID, opts.AllApproved)
	if err != nil {
		return repair.AuditSummary{}, repair.AuditSummary{}, err
	}

	currentRead, err := comics.NewSQLiteComicRepositoryReadOnly(opts.DBPath)
	if err != nil {
		return repair.AuditSummary{}, repair.AuditSummary{}, fmt.Errorf("open current db: %w", err)
	}
	currentAudit, err := currentRead.AuditDiscoveryIntegrity(ctx)
	_ = currentRead.Close()
	if err != nil {
		return repair.AuditSummary{}, repair.AuditSummary{}, fmt.Errorf("audit current db: %w", err)
	}

	backupRead, err := comics.NewSQLiteComicRepositoryReadOnly(opts.BackupPath)
	if err != nil {
		return repair.AuditSummary{}, repair.AuditSummary{}, fmt.Errorf("open backup db: %w", err)
	}
	backupAudit, err := backupRead.AuditDiscoveryIntegrity(ctx)
	_ = backupRead.Close()
	if err != nil {
		return repair.AuditSummary{}, repair.AuditSummary{}, fmt.Errorf("audit backup db: %w", err)
	}

	if plan.DatabaseFingerprint != currentAudit.DatabaseFingerprint ||
		plan.DatabaseFingerprint != backupAudit.DatabaseFingerprint {
		return repair.AuditSummary{}, repair.AuditSummary{}, fmt.Errorf("plan fingerprint does not match database or backup")
	}
	if err := repair.ValidatePlan(plan, currentAudit); err != nil {
		return repair.AuditSummary{}, repair.AuditSummary{}, err
	}

	before := currentAudit.Summary
	repo, err := comics.NewSQLiteComicRepository(opts.DBPath)
	if err != nil {
		return repair.AuditSummary{}, repair.AuditSummary{}, fmt.Errorf("open writable db: %w", err)
	}
	defer repo.Close()

	err = repo.WithTx(ctx, func(txRepo domain.ComicRepository) error {
		for _, group := range groups {
			if err := applyRepairGroup(ctx, txRepo, group); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return repair.AuditSummary{}, repair.AuditSummary{}, fmt.Errorf("apply repair tx: %w", err)
	}

	afterAudit, err := repo.AuditDiscoveryIntegrity(ctx)
	if err != nil {
		return repair.AuditSummary{}, repair.AuditSummary{}, fmt.Errorf("audit after apply: %w", err)
	}
	return before, afterAudit.Summary, nil
}

func applyRepairGroup(ctx context.Context, txRepo domain.ComicRepository, group repair.Group) error {
	switch group.Review.Decision {
	case repair.DecisionRestoreKeys:
		comic, err := txRepo.Get(ctx, group.Keyless.ID)
		if err != nil {
			return err
		}
		_, err = txRepo.Update(ctx, comic)
		return err
	case repair.DecisionMerge:
		service := usecase.NewComicServiceWithObserver(txRepo, nil, discovery.NoopObserver{})
		for _, mergeID := range group.Review.MergeIDs {
			if _, err := service.Merge(ctx, group.Review.CanonicalID, mergeID); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("group %s has non-applicable decision %q", group.GroupID, group.Review.Decision)
	}
}
