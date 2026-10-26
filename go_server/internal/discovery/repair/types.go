package repair

import "comics/internal/identity"

type ComicSnapshot struct {
	ID          int      `json:"id"`
	Titles      []string `json:"titles"`
	CurrentChap int      `json:"current_chap"`
	ComType     int      `json:"com_type"`
	Track       bool     `json:"track"`
	Deleted     bool     `json:"deleted"`
}

type KeyOwner struct {
	Key   identity.ScopedTitleKey `json:"key"`
	Comic ComicSnapshot           `json:"comic"`
}

type KeylessRecord struct {
	Comic            ComicSnapshot             `json:"comic"`
	ProposedKeys     []identity.ScopedTitleKey `json:"proposed_keys"`
	Owners           []KeyOwner                `json:"owners"`
	Classification   string                    `json:"classification"`
}

const (
	ClassificationActiveConflict = "active_conflict"
	ClassificationRestorable     = "restorable"
	ClassificationBlockedNoKey   = "blocked_no_key"
)

type AuditSummary struct {
	TotalComics     int `json:"total_comics"`
	ActiveComics    int `json:"active_comics"`
	KeyRows         int `json:"key_rows"`
	ActiveKeyOwners int `json:"active_key_owners"`
	KeylessActive   int `json:"keyless_active"`
	ActiveConflicts int `json:"active_conflicts"`
	Restorable      int `json:"restorable"`
	BlockedNoKey    int `json:"blocked_no_key"`
	DeletedKeyRows  int `json:"deleted_key_rows"`
	OrphanKeyRows   int `json:"orphan_key_rows"`
}

type Audit struct {
	DatabaseFingerprint string          `json:"database_fingerprint"`
	Summary             AuditSummary    `json:"summary"`
	Records             []KeylessRecord `json:"records"`
}
