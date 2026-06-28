package state

import (
	"os"
	"sort"
	"strings"
	"sync"
)

type EnvironmentStatus string

const (
	StatusActive    EnvironmentStatus = "active"
	StatusInactive  EnvironmentStatus = "inactive"
	StatusDeploying EnvironmentStatus = "deploying"
	StatusError     EnvironmentStatus = "error"
)

type EnvironmentState struct {
	OdooVersion string            `json:"odoo_version"`
	Branch      string            `json:"branch"`
	Status      EnvironmentStatus `json:"status"`
}

type Heartbeat struct {
	LastEventID      int                `json:"last_event_id"`
	RepoURL          string             `json:"repo_url"`
	ProductionBranch EnvironmentState   `json:"production_branch"`
	StagingBranches  []EnvironmentState `json:"staging_branches"`
	Backups          []string           `json:"backups"`
	WSUrl            string             `json:"ws_url"`
}

type AgentState struct {
	mu               sync.RWMutex
	lastEventID      int
	repoURL          string
	productionBranch EnvironmentState
	stagingBranches  []EnvironmentState
	backups          []string
	wsURL            string
}

func New(repoURL string) *AgentState {
	return &AgentState{
		repoURL:         repoURL,
		stagingBranches: []EnvironmentState{},
		backups:         []string{},
		wsURL:           buildWSUrl(),
	}
}

func buildWSUrl() string {
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		return ""
	}
	scheme := "ws://"
	if strings.HasPrefix(publicURL, "https://") {
		scheme = "wss://"
		publicURL = strings.TrimPrefix(publicURL, "https://")
	}
	publicURL = strings.TrimPrefix(publicURL, "http://")
	publicURL = strings.TrimRight(publicURL, "/")
	return scheme + publicURL
}

func (s *AgentState) BuildHeartbeat() Heartbeat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hb := Heartbeat{
		LastEventID:      s.lastEventID,
		RepoURL:          s.repoURL,
		ProductionBranch: s.productionBranch,
		StagingBranches:  make([]EnvironmentState, len(s.stagingBranches)),
		Backups:          make([]string, len(s.backups)),
		WSUrl:            s.wsURL,
	}
	copy(hb.StagingBranches, s.stagingBranches)
	copy(hb.Backups, s.backups)
	return hb
}

func (s *AgentState) Update(prod EnvironmentState, staging []EnvironmentState, backups []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.productionBranch = prod
	s.stagingBranches = staging
	s.backups = backups
	if s.backups == nil {
		s.backups = []string{}
	}
	sort.Strings(s.backups)
}

func (s *AgentState) SetLastEventID(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id > s.lastEventID {
		s.lastEventID = id
	}
}

func (s *AgentState) LastEventID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastEventID
}

func (s *AgentState) SetWSUrl(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wsURL = url
}

func (s *AgentState) AddStagingBranch(env EnvironmentState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.stagingBranches {
		if e.Branch == env.Branch {
			s.stagingBranches[i] = env
			return
		}
	}
	s.stagingBranches = append(s.stagingBranches, env)
}

func (s *AgentState) RemoveStagingBranch(branch string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.stagingBranches {
		if e.Branch == branch {
			s.stagingBranches = append(s.stagingBranches[:i], s.stagingBranches[i+1:]...)
			return
		}
	}
}

func (s *AgentState) SetProductionBranch(env EnvironmentState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.productionBranch = env
}

func (s *AgentState) GetProductionBranch() EnvironmentState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.productionBranch
}
