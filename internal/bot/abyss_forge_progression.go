package bot

import "database/sql"

func bumpAbyssForgeMilestone(tx *sql.Tx, uid, milestoneID string) (int64, error) {
	var progress int64
	err := tx.QueryRow(`INSERT INTO abyss_forge_milestones (client_uid, milestone_id, progress)
		VALUES ($1,$2,1)
		ON CONFLICT (client_uid, milestone_id) DO UPDATE
		SET progress=abyss_forge_milestones.progress+1, updated_at=NOW()
		RETURNING progress`, uid, milestoneID).Scan(&progress)
	return progress, err
}

func abyssForgeMilestoneStage(progress int64) string {
	switch {
	case progress >= 25:
		return "Exalted"
	case progress >= 10:
		return "Established"
	case progress >= 5:
		return "Proven"
	case progress >= 1:
		return "Marked"
	default:
		return "Unstarted"
	}
}
