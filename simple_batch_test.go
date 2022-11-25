package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSimpleBatcher_Config_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  SimpleBatcherConfig
		wantErr bool
	}{
		{name: "small batch interval", config: SimpleBatcherConfig{BatchIntervalMS: 0, BatchTimeoutMS: 1000}, wantErr: true},
		{name: "small batch timeout", config: SimpleBatcherConfig{BatchIntervalMS: 1000, BatchTimeoutMS: 1}, wantErr: true},
		{name: "valid config", config: SimpleBatcherConfig{BatchIntervalMS: 1000, BatchTimeoutMS: 1000}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("wanted error, but err == nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("didn't wanted error, but err == %v", err)
			}
		})
	}
}

var _ Job = new(shortJob)

type shortJob struct {
	doCount      atomic.Int32
	doneCount    atomic.Int32
	timeoutCount atomic.Int32
}

func (j *shortJob) do(ctx context.Context) {
	j.doCount.Add(1)
	time.Sleep(time.Duration(1000) * time.Millisecond)
}

func (j *shortJob) doOnDone(ctx context.Context) {
	j.doneCount.Add(1)
}
func (j *shortJob) doOnTimeout(ctx context.Context) {
	j.timeoutCount.Add(1)
}

func TestSimpleBatcher_Short_Job(t *testing.T) {
	ctx := context.Background()
	j := &shortJob{}
	batcher, err := NewSimpleBatcher(ctx, SimpleBatcherConfig{BatchTimeoutMS: 2000, BatchIntervalMS: 2000}, j)
	assert.Nil(t, err)
	err = batcher.Start(ctx)
	assert.Nil(t, err)
	time.Sleep(time.Duration(10) * time.Second)
	err = batcher.Stop(ctx)
	assert.Nil(t, err)
	assert.Equal(t, int32(0), j.timeoutCount.Load())
	assert.Equal(t, j.doCount.Load(), j.doneCount.Load())
}

var _ Job = new(longJob)

type longJob struct {
	doCount      atomic.Int32
	doneCount    atomic.Int32
	timeoutCount atomic.Int32
}

func (j *longJob) do(ctx context.Context) {
	j.doCount.Add(1)
	time.Sleep(time.Duration(2000) * time.Millisecond)
}

func (j *longJob) doOnDone(ctx context.Context) {
	j.doneCount.Add(1)
}

func (j *longJob) doOnTimeout(ctx context.Context) {
	j.timeoutCount.Add(1)
}
func TestSimpleBatcher_Long_Job(t *testing.T) {
	ctx := context.Background()
	j := &longJob{}
	batcher, err := NewSimpleBatcher(ctx, SimpleBatcherConfig{BatchTimeoutMS: 1000, BatchIntervalMS: 1500}, j)
	assert.Nil(t, err)
	err = batcher.Start(ctx)
	assert.Nil(t, err)
	time.Sleep(time.Duration(5) * time.Second)
	err = batcher.Stop(ctx)
	assert.Equal(t, int32(0), j.doneCount.Load())
	assert.Equal(t, j.doCount.Load(), j.timeoutCount.Load())
	assert.Nil(t, err)
}
