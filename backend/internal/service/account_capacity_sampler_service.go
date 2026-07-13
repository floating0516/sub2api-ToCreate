package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

const (
	accountCapacitySamplerInterval  = time.Minute
	accountCapacitySamplerTimeout   = 8 * time.Second
	accountCapacitySamplerBatchSize = 200
)

type AccountCapacitySampleRepository interface {
	InsertSamples(ctx context.Context, samples []usagestats.AccountCapacitySample) error
}

type AccountCapacitySamplerService struct {
	accountRepo        AccountRepository
	concurrencyService *ConcurrencyService
	sampleRepo         AccountCapacitySampleRepository
	interval           time.Duration

	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
}

func NewAccountCapacitySamplerService(
	accountRepo AccountRepository,
	concurrencyService *ConcurrencyService,
	sampleRepo AccountCapacitySampleRepository,
	interval time.Duration,
) *AccountCapacitySamplerService {
	if interval <= 0 {
		interval = accountCapacitySamplerInterval
	}
	return &AccountCapacitySamplerService{
		accountRepo:        accountRepo,
		concurrencyService: concurrencyService,
		sampleRepo:         sampleRepo,
		interval:           interval,
	}
}

func (s *AccountCapacitySamplerService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		if s.stopCh == nil {
			s.stopCh = make(chan struct{})
		}
		go s.run()
	})
}

func (s *AccountCapacitySamplerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
		}
	})
}

func (s *AccountCapacitySamplerService) run() {
	s.sampleOnce()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.sampleOnce()
		case <-s.stopCh:
			return
		}
	}
}

func (s *AccountCapacitySamplerService) sampleOnce() {
	if s == nil || s.accountRepo == nil || s.concurrencyService == nil || s.sampleRepo == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), accountCapacitySamplerTimeout)
	defer cancel()

	accounts, err := s.accountRepo.ListActive(ctx)
	if err != nil {
		logger.LegacyPrintf("service.account_capacity_sampler", "list active accounts failed: %v", err)
		return
	}
	if len(accounts) == 0 {
		return
	}

	sampledAt := time.Now().UTC()
	samples := make([]usagestats.AccountCapacitySample, 0, len(accounts))

	for start := 0; start < len(accounts); start += accountCapacitySamplerBatchSize {
		end := start + accountCapacitySamplerBatchSize
		if end > len(accounts) {
			end = len(accounts)
		}

		loadReq := make([]AccountWithConcurrency, 0, end-start)
		maxByID := make(map[int64]int64, end-start)
		for i := start; i < end; i++ {
			acc := accounts[i]
			if acc.ID <= 0 {
				continue
			}
			maxConcurrency := acc.EffectiveLoadFactor()
			loadReq = append(loadReq, AccountWithConcurrency{
				ID:             acc.ID,
				MaxConcurrency: maxConcurrency,
			})
			maxByID[acc.ID] = int64(maxConcurrency)
		}
		if len(loadReq) == 0 {
			continue
		}

		loadMap, loadErr := s.concurrencyService.GetAccountsLoadBatch(ctx, loadReq)
		if loadErr != nil {
			logger.LegacyPrintf("service.account_capacity_sampler", "get account load batch failed: %v", loadErr)
			continue
		}

		for _, req := range loadReq {
			load := loadMap[req.ID]
			sample := usagestats.AccountCapacitySample{
				AccountID:      req.ID,
				SampledAt:      sampledAt,
				MaxConcurrency: maxByID[req.ID],
			}
			if load != nil {
				sample.CurrentConcurrency = int64(load.CurrentConcurrency)
				sample.WaitingCount = int64(load.WaitingCount)
			}
			samples = append(samples, sample)
		}
	}

	if len(samples) == 0 {
		return
	}
	if err := s.sampleRepo.InsertSamples(ctx, samples); err != nil {
		logger.LegacyPrintf("service.account_capacity_sampler", "insert account capacity samples failed: %v", err)
	}
}
