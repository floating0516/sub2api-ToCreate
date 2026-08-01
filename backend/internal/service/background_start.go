package service

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const backgroundStartDelayEnv = "SERVER_BACKGROUND_START_DELAY_SECONDS"

var delayedBackgroundStarts struct {
	once   sync.Once
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func parseBackgroundStartDelay(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 || seconds > 3600 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func backgroundStartContext() context.Context {
	delayedBackgroundStarts.once.Do(func() {
		delayedBackgroundStarts.ctx, delayedBackgroundStarts.cancel = context.WithCancel(context.Background())
	})
	return delayedBackgroundStarts.ctx
}

func scheduleBackgroundStart(name string, start func()) {
	if start == nil {
		return
	}

	rawDelay := os.Getenv(backgroundStartDelayEnv)
	delay := parseBackgroundStartDelay(rawDelay)
	if strings.TrimSpace(rawDelay) != "" && delay == 0 && strings.TrimSpace(rawDelay) != "0" {
		logger.LegacyPrintf("service.background_start", "[%s] invalid %s=%q; starting immediately", name, backgroundStartDelayEnv, rawDelay)
	}
	if delay <= 0 {
		start()
		return
	}

	ctx := backgroundStartContext()
	delayedBackgroundStarts.wg.Add(1)
	go func() {
		defer delayedBackgroundStarts.wg.Done()
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			logger.LegacyPrintf("service.background_start", "[%s] delayed start canceled", name)
		case <-timer.C:
			logger.LegacyPrintf("service.background_start", "[%s] starting after %s delay", name, delay)
			start()
		}
	}()
}

// CancelDelayedBackgroundStarts prevents a draining instance from starting a
// scheduler after shutdown has begun.
func CancelDelayedBackgroundStarts() {
	if delayedBackgroundStarts.cancel != nil {
		delayedBackgroundStarts.cancel()
	}
	delayedBackgroundStarts.wg.Wait()
}
