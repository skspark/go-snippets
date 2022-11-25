package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Job interface {
	do(ctx context.Context)
	doOnTimeout(ctx context.Context)
	doOnDone(ctx context.Context)
}

type Batcher interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

var _ Batcher = new(SimpleBatcher)

type SimpleBatcherConfig struct {
	BatchIntervalMS int64 `json:"batch_interval_ms"`
	BatchTimeoutMS  int64 `json:"batch_timeout_ms"`
}

func (c SimpleBatcherConfig) Validate() error {
	if c.BatchIntervalMS <= 1 {
		return fmt.Errorf("batch interval ms:%d is invalid", c.BatchIntervalMS)
	}
	if c.BatchTimeoutMS <= 1 {
		return fmt.Errorf("batch timeout ms:%d is invalid", c.BatchTimeoutMS)
	}
	return nil
}

type SimpleBatcher struct {
	config           SimpleBatcherConfig
	waitGroup        sync.WaitGroup
	ticker           *time.Ticker
	tickerFinishChan chan struct{}
	job              Job
}

func NewSimpleBatcher(ctx context.Context, config SimpleBatcherConfig, job Job) (Batcher, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &SimpleBatcher{
		config:           config,
		ticker:           time.NewTicker(time.Duration(config.BatchIntervalMS) * time.Millisecond),
		tickerFinishChan: make(chan struct{}, 1),
		job:              job,
	}, nil
}

func (b *SimpleBatcher) Start(ctx context.Context) error {
	go func() {
		for {
			select {
			case <-b.tickerFinishChan:
				return
			case <-b.ticker.C:
				b.run(ctx)
			}
		}
	}()
	return nil
}

func (b *SimpleBatcher) run(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(b.config.BatchTimeoutMS)*time.Millisecond)
	b.waitGroup.Add(1)
	defer func() {
		println("HIHI Defer")
		if err := recover(); err != nil {
			println(err)
		}
		cancel()
		b.waitGroup.Done()
	}()
	jobDoneChan := make(chan struct{}, 1)
	go func(ctx context.Context) {
		defer func() {
			jobDoneChan <- struct{}{}
		}()
		b.job.do(ctx)
	}(ctx)
	select {
	case <-ctx.Done():
		b.job.doOnTimeout(ctx)
	case <-jobDoneChan:
		b.job.doOnDone(ctx)
	}
}

func (s *SimpleBatcher) Stop(ctx context.Context) error {
	s.ticker.Stop()
	s.tickerFinishChan <- struct{}{}
	s.waitGroup.Wait()
	return nil
}
