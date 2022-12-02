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
		tickerFinishChan: make(chan struct{}),
		job:              job,
	}, nil
}

func (b *SimpleBatcher) Start(ctx context.Context) error {
	go func() {
		print(b.tickerFinishChan)
		for {
			select {
			case _ = <-b.ticker.C:
				b.run(context.Background())
			case _ = <-b.tickerFinishChan:
				return
			}
		}
	}()
	return nil
}

func (b *SimpleBatcher) run(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(b.config.BatchTimeoutMS)*time.Millisecond)
	b.waitGroup.Add(1)
	defer func() {
		if err := recover(); err != nil {
			println(err)
		}
		b.waitGroup.Done()
		cancel()
	}()
	jobDoneChan := make(chan struct{})
	go func(ctx context.Context) {
		defer func() {
			jobDoneChan <- struct{}{}
		}()
		b.job.do(ctx)
	}(ctx)
	select {
	case _ = <-ctx.Done():
		b.job.doOnTimeout(ctx)
	case _ = <-jobDoneChan:
		b.job.doOnDone(ctx)
	}
}

func (b *SimpleBatcher) Stop(ctx context.Context) error {
	b.ticker.Stop()
	b.waitGroup.Wait()
	b.tickerFinishChan <- struct{}{}
	return nil
}