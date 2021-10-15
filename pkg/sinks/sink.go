package sinks

import (
	"context"

	"github.com/champly/eventexporter/pkg/kube"
)

// Sink is the interface that the third-party providers should implement. It should just get the event and
// transform it depending on its configuration and submit it. Error handling for retries etc. should be handled inside
// for now.
type Sink interface {
	Send(ctx context.Context, ev *kube.EnhancedEvent) error
	Close()
}

// build sink func
type NewSinkFunc func(cfg interface{}) (Sink, error)
