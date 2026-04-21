// Package datastream provides Kafka-to-analytics ETL pipeline.
// Phase 4 — Data Intelligence Layer.
package datastream

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/geocore-next/backend/pkg/kafka"
)

// TransformFunc converts a raw Kafka event into an analytics record.
type TransformFunc func(event kafka.Event) (interface{}, error)

// Pipeline processes events from Kafka and feeds them into an analytics sink.
type Pipeline struct {
	mu            sync.Mutex
	source        string // Kafka topic
	groupID       string
	transforms    []TransformFunc
	sink          func(ctx context.Context, records []interface{}) error
	batchSize     int
	flushInterval time.Duration
}

// NewPipeline creates an ETL pipeline for a Kafka topic.
func NewPipeline(source, groupID string, opts ...PipelineOption) *Pipeline {
	p := &Pipeline{
		source:        source,
		groupID:       groupID,
		batchSize:     100,
		flushInterval: 5 * time.Second,
	}
	p.sink = func(ctx context.Context, records []interface{}) error {
		slog.Debug("datastream: flushing batch", "source", source, "count", len(records))
		return nil
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// PipelineOption configures a pipeline.
type PipelineOption func(*Pipeline)

// WithBatchSize sets the batch size for the pipeline.
func WithBatchSize(n int) PipelineOption {
	return func(p *Pipeline) { p.batchSize = n }
}

// WithFlushInterval sets how often to flush partial batches.
func WithFlushInterval(d time.Duration) PipelineOption {
	return func(p *Pipeline) { p.flushInterval = d }
}

// WithSink sets the output sink function.
func WithSink(fn func(ctx context.Context, records []interface{}) error) PipelineOption {
	return func(p *Pipeline) { p.sink = fn }
}

// AddTransform adds a transformation step to the pipeline.
func (p *Pipeline) AddTransform(fn TransformFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.transforms = append(p.transforms, fn)
}

// Process applies all transforms to an event and returns the result.
func (p *Pipeline) Process(ctx context.Context, event kafka.Event) (interface{}, error) {
	p.mu.Lock()
	transforms := p.transforms
	p.mu.Unlock()

	var result interface{} = event
	for _, fn := range transforms {
		r, err := fn(event)
		if err != nil {
			return nil, err
		}
		result = r
	}
	return result, nil
}
