package agent

import (
	"context"
	"fmt"

	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/pkg/types"
	"github.com/google/uuid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const (
	DefaultTaskTopic   = "betar/tasks"
	DefaultResultTopic = "betar/tasks/results"
)

// Worker consumes task requests from pubsub and publishes task results.
type Worker struct {
	pubsub      *p2p.PubSub
	runtime     Runtime
	taskTopic   string
	resultTopic string
}

// NewWorker creates a pubsub worker.
func NewWorker(ps *p2p.PubSub, runtime Runtime, taskTopic, resultTopic string) *Worker {
	if taskTopic == "" {
		taskTopic = DefaultTaskTopic
	}
	if resultTopic == "" {
		resultTopic = DefaultResultTopic
	}

	return &Worker{
		pubsub:      ps,
		runtime:     runtime,
		taskTopic:   taskTopic,
		resultTopic: resultTopic,
	}
}

// Start subscribes and processes incoming task messages until context cancellation.
func (w *Worker) Start(ctx context.Context) error {
	if w.pubsub == nil {
		return fmt.Errorf("pubsub is required")
	}
	if w.runtime == nil {
		return fmt.Errorf("runtime is required")
	}

	sub, err := w.pubsub.Subscribe(ctx, w.taskTopic)
	if err != nil {
		return fmt.Errorf("failed to subscribe to task topic: %w", err)
	}
	defer sub.Cancel()

	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("failed to read task message: %w", err)
		}

		if err := w.HandleMessage(ctx, msg); err != nil {
			continue
		}
	}
}

// HandleMessage handles a single pubsub task message.
func (w *Worker) HandleMessage(ctx context.Context, msg *pubsub.Message) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	var req types.TaskRequest
	if err := types.FromJSON(msg.Data, &req); err != nil {
		return fmt.Errorf("failed to decode task request: %w", err)
	}

	if req.RequestID == "" {
		req.RequestID = uuid.NewString()
	}

	result, runErr := w.runtime.RunTask(ctx, req)
	resp := types.TaskResponse{RequestID: req.RequestID}

	if runErr != nil {
		resp.Error = runErr.Error()
	} else {
		resp.Output = result.Output
		if result.Error != "" {
			resp.Error = result.Error
		}
	}

	payload, err := types.ToJSON(resp)
	if err != nil {
		return fmt.Errorf("failed to encode task response: %w", err)
	}

	if err := w.pubsub.Publish(ctx, w.resultTopic, payload); err != nil {
		return fmt.Errorf("failed to publish task response: %w", err)
	}

	return nil
}
