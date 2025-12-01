package api

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Proposal struct {
	ID          uint64    `json:"id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type proposalStore struct {
	mu        sync.RWMutex
	counter   atomic.Uint64
	proposals map[uint64]Proposal
}

func newProposalStore() *proposalStore {
	store := &proposalStore{proposals: make(map[uint64]Proposal)}
	store.seed()
	return store
}

func (ps *proposalStore) seed() {
	mockProposals := []string{
		"Enable staking rewards redistribution",
		"Launch developer grants program",
		"Reduce transaction fees by 10%",
	}

	for _, description := range mockProposals {
		ps.CreateProposal(description)
	}
}

func (ps *proposalStore) CreateProposal(description string) Proposal {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now().UTC()
	id := ps.counter.Add(1)
	proposal := Proposal{
		ID:          id,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	ps.proposals[id] = proposal
	return proposal
}

func (ps *proposalStore) ListProposals() []Proposal {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	proposals := make([]Proposal, 0, len(ps.proposals))
	for _, p := range ps.proposals {
		proposals = append(proposals, p)
	}

	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].ID < proposals[j].ID
	})

	return proposals
}

func (ps *proposalStore) GetProposal(id uint64) (Proposal, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	proposal, ok := ps.proposals[id]
	return proposal, ok
}
