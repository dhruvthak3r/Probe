package mq

import (
	"context"
	"encoding/json"
	"fmt"
	//"github.com/dhruvthak3r/Probe/internal/monitor"
)

func (c *Consumer) ConsumeFromQueue(ctx context.Context) error {

	mssgs, err := c.ch.Consume(c.queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error consuming from rabbitmq: %v", err)
	}

	for {
		select {
		case m, ok := <-mssgs:
			if !ok {
				return fmt.Errorf("message channel closed")
			}

			res := ResultMessage{}

			err := json.Unmarshal(m.Body, &res)
			if err != nil {
				return fmt.Errorf("failed to unmarshal message body: %v", err)
			}

		case <-ctx.Done():
			return fmt.Errorf("stopping consumer %v", ctx.Err())
		}
	}
}
