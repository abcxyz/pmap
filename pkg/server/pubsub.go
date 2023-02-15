// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
)

// PubSubMessenger implements the Messenger interface for Google Cloud PubSub.
type PubSubMessenger struct {
	client *pubsub.Client
	topic  *pubsub.Topic
}

// MessengerOption is the option to set up a PubSubMessenger.
type MessengerOption func(p *PubSubMessenger) (*PubSubMessenger, error)

// WithClient provides a PubSub client to the PubSubMessenger.
func WithClient(client *pubsub.Client) MessengerOption {
	return func(p *PubSubMessenger) (*PubSubMessenger, error) {
		p.client = client
		return p, nil
	}
}

// NewPubSubMessenger creates a new instance of the PubSubMessenger.
// 
// The project ID will be used to create a PubSub client if a client is not provided.
// The topic ID is the PubSub topic name in the client's project.
func NewPubSubMessenger(ctx context.Context, projectID, topicID string, opts ...MessengerOption) (*PubSubMessenger, error) {
	p := &PubSubMessenger{}
	for _, opt := range opts {
		var err error
		p, err = opt(p)
		if err != nil {
			return nil, fmt.Errorf("failed to apply messenger options: %w", err)
		}
	}
	if p.client == nil {
		client, err := pubsub.NewClient(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to create new PubSub client: %w", err)
		}
		p.client = client
	}

	p.topic = p.client.Topic(topicID)

	return p, nil
}

// Send sends a message to a Google Cloud PubSub topic.
func (p *PubSubMessenger) Send(ctx context.Context, msg []byte) error {
	result := p.topic.Publish(ctx, &pubsub.Message{
		Data: msg,
	})

	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("pubsub: failed to get result returned from publish : %w", err)
	}
	return nil
}

// Cleanup handles the graceful shutdown of the PubSub client.
func (p *PubSubMessenger) Cleanup() error {
	p.topic.Stop()
	if err := p.client.Close(); err != nil {
		return fmt.Errorf("failed to close PubSub client: %w", err)
	}
	return nil
}
