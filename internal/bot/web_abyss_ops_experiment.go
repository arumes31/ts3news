package bot

const abyssExperimentGuardrailSample = 20

type abyssRewardCohortMetrics struct {
	Floors       int64 `json:"floors"`
	CombatFloors int64 `json:"combat_floors"`
	Wins         int64 `json:"wins"`
	Deaths       int64 `json:"deaths"`
	Reward       int64 `json:"reward"`
	Anomalies    int64 `json:"anomalies"`
}

type abyssRewardCohortSnapshot struct {
	abyssRewardCohortMetrics
	DeathRate     float64 `json:"death_rate"`
	AnomalyRate   float64 `json:"anomaly_rate"`
	AverageReward float64 `json:"average_reward"`
}

type abyssRewardExperimentSnapshot struct {
	Enabled  bool                                 `json:"enabled"`
	Revision uint64                               `json:"revision"`
	Status   string                               `json:"status"`
	Cohorts  map[string]abyssRewardCohortSnapshot `json:"cohorts"`
}

func (m *abyssOpsMetrics) resetRewardExperiment(revision uint64) {
	m.mu.Lock()
	m.rewardExperimentRevision = revision
	m.rewardCohorts = make(map[string]abyssRewardCohortMetrics)
	m.mu.Unlock()
}

func (m *abyssOpsMetrics) observeRewardExperiment(assignment abyssRewardAssignment, bonus int64, combat, victory, anomaly bool) {
	if assignment.Cohort == "off" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rewardExperimentRevision == 0 {
		m.rewardExperimentRevision = assignment.Revision
	}
	if assignment.Revision != m.rewardExperimentRevision {
		return
	}
	if m.rewardCohorts == nil {
		m.rewardCohorts = make(map[string]abyssRewardCohortMetrics)
	}
	cohort := m.rewardCohorts[assignment.Cohort]
	cohort.Floors++
	cohort.Reward += bonus
	if combat {
		cohort.CombatFloors++
		if victory {
			cohort.Wins++
		} else {
			cohort.Deaths++
		}
	}
	if anomaly {
		cohort.Anomalies++
	}
	m.rewardCohorts[assignment.Cohort] = cohort
}

func (m *abyssOpsMetrics) rewardExperimentSnapshot(features abyssFeatureSnapshot) abyssRewardExperimentSnapshot {
	m.mu.Lock()
	cohorts := make(map[string]abyssRewardCohortSnapshot, len(m.rewardCohorts))
	for name, metrics := range m.rewardCohorts {
		cohorts[name] = abyssRewardCohortSnapshot{
			abyssRewardCohortMetrics: metrics,
			DeathRate:                ratio(metrics.Deaths, metrics.CombatFloors),
			AnomalyRate:              ratio(metrics.Anomalies, metrics.Floors),
			AverageReward:            ratio(metrics.Reward, metrics.Floors),
		}
	}
	m.mu.Unlock()
	status := "disabled"
	if features.RewardExperimentEnabled {
		status = "collecting"
		control, controlOK := cohorts["control"]
		treatment, treatmentOK := cohorts["treatment"]
		if controlOK && treatmentOK && control.CombatFloors >= abyssExperimentGuardrailSample && treatment.CombatFloors >= abyssExperimentGuardrailSample {
			status = "healthy"
			if treatment.DeathRate > control.DeathRate+0.05 || treatment.AnomalyRate > 0.01 {
				status = "halt_recommended"
			}
		}
	}
	return abyssRewardExperimentSnapshot{
		Enabled:  features.RewardExperimentEnabled,
		Revision: features.RewardExperimentRevision,
		Status:   status,
		Cohorts:  cohorts,
	}
}
