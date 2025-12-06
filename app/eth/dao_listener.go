package eth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"web3-backend/app/api"
)

const (
	proposalCreatedEventName  = "ProposalCreated"
	votedEventName            = "Voted"
	proposalExecutedEventName = "ProposalExecuted"

	proposalStatusExecuted = 2
)

// DAOEventListener subscribes to DAOContract events and mirrors them in memory.
type DAOEventListener struct {
	client   *ethclient.Client
	address  common.Address
	abi      abi.ABI
	store    *api.ProposalStore
	logger   *log.Logger
	interval time.Duration
}

// NewDAOEventListener wires DAO events to the in-memory proposal store.
func NewDAOEventListener(rpcURL, contractAddress, abiPath string, store *api.ProposalStore, logger *log.Logger) (*DAOEventListener, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial eth client: %w", err)
	}

	abiBytes, err := os.ReadFile(abiPath)
	if err != nil {
		return nil, fmt.Errorf("read abi: %w", err)
	}

	parsedABI, err := abi.JSON(bytes.NewReader(abiBytes))
	if err != nil {
		return nil, fmt.Errorf("parse abi: %w", err)
	}

	return &DAOEventListener{
		client:   client,
		address:  common.HexToAddress(contractAddress),
		abi:      parsedABI,
		store:    store,
		logger:   logger,
		interval: 5 * time.Second,
	}, nil
}

// Run blocks while streaming new DAO events until the context is canceled.
func (l *DAOEventListener) Run(ctx context.Context) error {
	defer l.client.Close()

	if err := l.runSubscription(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		if isNotificationUnsupported(err) {
			l.logger.Printf("log notifications unsupported, falling back to polling: %v", err)
			return l.runPolling(ctx)
		}
		return err
	}

	return nil
}

func (l *DAOEventListener) runSubscription(ctx context.Context) error {
	query := ethereum.FilterQuery{Addresses: []common.Address{l.address}}
	logs := make(chan types.Log)
	sub, err := l.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return fmt.Errorf("subscribe logs: %w", err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-sub.Err():
			if err != nil {
				return fmt.Errorf("subscription error: %w", err)
			}
		case vLog := <-logs:
			if err := l.dispatchLog(vLog); err != nil {
				l.logger.Printf("failed to handle dao log: %v", err)
			}
		}
	}
}

func (l *DAOEventListener) runPolling(ctx context.Context) error {
	latestBlock, err := l.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("fetch latest block: %w", err)
	}
	lastProcessed := latestBlock
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			latest, err := l.client.BlockNumber(ctx)
			if err != nil {
				l.logger.Printf("poll error: %v", err)
				continue
			}
			if latest <= lastProcessed {
				continue
			}
			from := lastProcessed + 1
			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(from)),
				ToBlock:   big.NewInt(int64(latest)),
				Addresses: []common.Address{l.address},
			}

			logs, err := l.client.FilterLogs(ctx, query)
			if err != nil {
				l.logger.Printf("filter logs error: %v", err)
				continue
			}
			for _, vLog := range logs {
				if err := l.dispatchLog(vLog); err != nil {
					l.logger.Printf("failed to handle dao log (polling): %v", err)
				}
			}
			lastProcessed = latest
		}
	}
}

func isNotificationUnsupported(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "notifications not supported")
}

func (l *DAOEventListener) dispatchLog(vLog types.Log) error {
	if len(vLog.Topics) == 0 {
		return fmt.Errorf("log missing topics")
	}

	event, err := l.abi.EventByID(vLog.Topics[0])
	if err != nil {
		return fmt.Errorf("unknown event id: %w", err)
	}

	switch event.Name {
	case proposalCreatedEventName:
		return l.handleProposalCreated(vLog)
	case votedEventName:
		return l.handleVoted(vLog)
	case proposalExecutedEventName:
		return l.handleProposalExecuted(vLog)
	default:
		return nil
	}
}

func (l *DAOEventListener) handleProposalCreated(vLog types.Log) error {
	if len(vLog.Topics) < 3 {
		return fmt.Errorf("proposal created log missing topics")
	}

	var eventData struct {
		Description string
	}

	if err := l.abi.UnpackIntoInterface(&eventData, proposalCreatedEventName, vLog.Data); err != nil {
		return fmt.Errorf("unpack proposal created: %w", err)
	}

	id, err := topicToUint64(vLog.Topics[1])
	if err != nil {
		return fmt.Errorf("parse proposal id: %w", err)
	}

	creator := topicToAddress(vLog.Topics[2])

	l.store.HandleProposalCreated(id, eventData.Description, creator.Hex(), time.Now().UTC())
	l.logger.Printf("ProposalCreated id=%d description=%s creator=%s", id, eventData.Description, creator.Hex())
	return nil
}

func (l *DAOEventListener) handleVoted(vLog types.Log) error {
	if len(vLog.Topics) < 3 {
		return fmt.Errorf("vote log missing topics")
	}

	var eventData struct {
		Support      bool
		ForVotes     *big.Int
		AgainstVotes *big.Int
		Status       uint8
	}

	if err := l.abi.UnpackIntoInterface(&eventData, votedEventName, vLog.Data); err != nil {
		return fmt.Errorf("unpack voted: %w", err)
	}

	id, err := topicToUint64(vLog.Topics[1])
	if err != nil {
		return fmt.Errorf("parse vote proposal id: %w", err)
	}

	voter := topicToAddress(vLog.Topics[2])
	supportVotes := bigToUint64(eventData.ForVotes)
	againstVotes := bigToUint64(eventData.AgainstVotes)
	executed := eventData.Status == proposalStatusExecuted

	l.store.HandleVote(id, eventData.Support, voter.Hex(), supportVotes, againstVotes, executed, time.Now().UTC())
	l.logger.Printf("Voted id=%d support=%t voter=%s", id, eventData.Support, voter.Hex())
	return nil
}

func (l *DAOEventListener) handleProposalExecuted(vLog types.Log) error {
	if len(vLog.Topics) < 3 {
		return fmt.Errorf("proposal executed log missing topics")
	}

	id, err := topicToUint64(vLog.Topics[1])
	if err != nil {
		return fmt.Errorf("parse executed proposal id: %w", err)
	}

	executor := topicToAddress(vLog.Topics[2])

	l.store.HandleProposalExecuted(id, executor.Hex(), time.Now().UTC())
	l.logger.Printf("ProposalExecuted id=%d executor=%s", id, executor.Hex())
	return nil
}

func topicToUint64(topic common.Hash) (uint64, error) {
	val := new(big.Int).SetBytes(topic[:])
	if !val.IsUint64() {
		return 0, fmt.Errorf("topic value %s exceeds uint64", val.String())
	}
	return val.Uint64(), nil
}

func topicToAddress(topic common.Hash) common.Address {
	return common.BytesToAddress(topic[12:])
}

func bigToUint64(value *big.Int) uint64 {
	if value == nil {
		return 0
	}
	if !value.IsUint64() {
		return math.MaxUint64
	}
	return value.Uint64()
}
