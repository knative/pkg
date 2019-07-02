/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package subscriber

import (
	"context"
	"encoding/json"
	"log"

	"cloud.google.com/go/pubsub"
	"github.com/knative/test-infra/tools/monitoring/prowapi"
)

// Client is a wrapper on the subscriber Operation
type Client struct {
	Operation
}

// Operation defines a list of methods for subscribing messages
type Operation interface {
	Receive(ctx context.Context, f func(context.Context, *pubsub.Message)) error
	String() string
}

// NewSubscriberClient returns a new SubscriberClient used to read crier pubsub messages
func NewSubscriberClient(ctx context.Context, projectID string, subName string) (*Client, error) {
	c, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &Client{c.Subscription(subName)}, nil
}

// ReceiveMessageAckAll acknowledges all incoming pusub messages and convert the pubsub message to ReportMessage.
// It executes `f` only if the pubsub message can be converted to ReportMessage. Otherwise, ignore the message.
func (c *Client) ReceiveMessageAckAll(ctx context.Context, f func(*prowapi.ReportMessage)) error {
	return c.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		if rmsg, err := c.toReportMessage(msg); err != nil {
			log.Printf("Cannot convert pubsub message (%v) to Report message %v", msg, err)
		} else if rmsg != nil {
			f(rmsg)
		}
		msg.Ack()
	})
}

func (c *Client) toReportMessage(msg *pubsub.Message) (*prowapi.ReportMessage, error) {
	rmsg := &prowapi.ReportMessage{}
	if err := json.Unmarshal(msg.Data, rmsg); err != nil {
		return nil, err
	}
	return rmsg, nil
}
