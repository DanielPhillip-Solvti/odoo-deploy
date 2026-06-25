package heartbeat

type EnvironmentStatus string

const (
	EnvironmentStatusActive    EnvironmentStatus = "active"
	EnvironmentStatusInactive  EnvironmentStatus = "inactive"
	EnvironmentStatusDeploying EnvironmentStatus = "deploying"
	EnvironmentStatusError     EnvironmentStatus = "error"
)

type EnvironmentState struct {
	Branch string            `json:"branch"`
	Status EnvironmentStatus `json:"status"`
}

type Heartbeat struct {
	LastEventID          int              `json:"last_event_id"`
	RepoURL              string           `json:"repo_url"`
	ProductionBranch     EnvironmentState `json:"production_branch"`
	StagingBranches      []EnvironmentState `json:"staging_branches"`
}

func BuildHeartbeat() Heartbeat {
	// TODO: implement
	// check local services on docker to build accurate heartbeat
	return Heartbeat{
		LastEventID: 0,
		RepoURL:     "",
		ProductionBranch: EnvironmentState{
			Branch: "main",
			Status: EnvironmentStatusActive,
		},
		StagingBranches: []EnvironmentState{},
	}
}
	