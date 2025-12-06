package api

import (
	"sort"
	"sync"
	"time"
)

// Proposal models a DAO proposal stored in-memory.
type Proposal struct {
	ID            uint64    `json:"id"`
	Description   string    `json:"description"`
	Proposer      string    `json:"proposer"`
	SupportVotes  uint64    `json:"supportVotes"`
	AgainstVotes  uint64    `json:"againstVotes"`
	Executed      bool      `json:"executed"`
	LastSupport   bool      `json:"lastSupport"`
	LastVoter     string    `json:"lastVoter"`
	Executor      string    `json:"executor"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	LastUpdatedBy string    `json:"lastUpdatedBy"`
}

// ProposalStore keeps proposals in memory with simple read/write locking.
type ProposalStore struct {
	mu        sync.RWMutex
	proposals map[uint64]Proposal
}

// NewProposalStore constructs an empty store populated via blockchain events.
func NewProposalStore() *ProposalStore {
	return &ProposalStore{proposals: make(map[uint64]Proposal)}
}

// HandleProposalCreated upserts a proposal originating from the DAO contract.
func (ps *ProposalStore) HandleProposalCreated(id uint64, description, creator string, observedAt time.Time) Proposal {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	proposal, exists := ps.proposals[id]
	proposal.ID = id
	proposal.Description = description
	proposal.Proposer = creator
	proposal.LastUpdatedBy = creator
	proposal.Executor = ""
	proposal.Executed = false
	proposal.LastVoter = ""
	proposal.LastSupport = false
	proposal.SupportVotes = 0
	proposal.AgainstVotes = 0
	proposal.UpdatedAt = observedAt
	if !exists || proposal.CreatedAt.IsZero() {
		proposal.CreatedAt = observedAt
	}

	ps.proposals[id] = proposal
	return proposal
}

// HandleVote records a vote and updates vote counters for a proposal.
func (ps *ProposalStore) HandleVote(
	id uint64,
	support bool,
	voter string,
	supportVotes uint64,
	againstVotes uint64,
	executed bool,
	observedAt time.Time,
) Proposal {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	proposal := ps.ensureProposalExists(id, observedAt)
	proposal.SupportVotes = supportVotes
	proposal.AgainstVotes = againstVotes
	proposal.LastSupport = support
	proposal.LastVoter = voter
	proposal.LastUpdatedBy = voter
	proposal.Executed = executed
	if !executed {
		proposal.Executor = ""
	}
	proposal.UpdatedAt = observedAt
	ps.proposals[id] = proposal
	return proposal
}

// HandleProposalExecuted marks a proposal as executed and stores executor info.
func (ps *ProposalStore) HandleProposalExecuted(id uint64, executor string, observedAt time.Time) Proposal {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	proposal := ps.ensureProposalExists(id, observedAt)
	proposal.Executed = true
	proposal.Executor = executor
	proposal.LastUpdatedBy = executor
	proposal.UpdatedAt = observedAt
	ps.proposals[id] = proposal
	return proposal
}

func (ps *ProposalStore) ensureProposalExists(id uint64, observedAt time.Time) Proposal {
	proposal, exists := ps.proposals[id]
	if !exists {
		proposal = Proposal{ID: id, CreatedAt: observedAt}
	} else if proposal.CreatedAt.IsZero() {
		proposal.CreatedAt = observedAt
	}
	return proposal
}

// ListProposals returns proposals sorted by ID.
func (ps *ProposalStore) ListProposals() []Proposal {
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

// GetProposal fetches a proposal by ID.
func (ps *ProposalStore) GetProposal(id uint64) (Proposal, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	proposal, ok := ps.proposals[id]
	return proposal, ok
}
