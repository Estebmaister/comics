package repair

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func WritePlanFile(path string, plan Plan) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(plan); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return nil
}

func LoadPlanFile(path string) (Plan, error) {
	file, err := os.Open(path)
	if err != nil {
		return Plan{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var plan Plan
	if err := decoder.Decode(&plan); err != nil {
		return Plan{}, err
	}
	if plan.SchemaVersion != PlanSchemaVersion {
		return Plan{}, fmt.Errorf("unsupported plan schema %d", plan.SchemaVersion)
	}
	return plan, nil
}

func PlanStats(plan Plan) (approved, merge, restore, blocked int) {
	for _, group := range plan.Groups {
		if !group.Review.Approved {
			if group.Risk == "blocked" {
				blocked++
			}
			continue
		}
		approved++
		switch group.Review.Decision {
		case DecisionMerge:
			merge++
		case DecisionRestoreKeys:
			restore++
		case DecisionBlock:
			blocked++
		}
	}
	return approved, merge, restore, blocked
}
