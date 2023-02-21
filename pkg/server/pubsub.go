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
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
)

// PubSubMessenger implements the Messenger interface for Google Cloud PubSub.
type PubSubMessenger struct {
	client *pubsub.Client
	topic  *pubsub.Topic
}

// NewPubSubMessenger creates a new instance of the PubSubMessenger.
func NewPubSubMessenger(ctx context.Context, projectID, topicID string, opts ...option.ClientOption) (*PubSubMessenger, error) {
	client, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create new pubsub client: %w", err)
	}

	topic := client.Topic(topicID)

	return &PubSubMessenger{client: client, topic: topic}, nil
}

// Send sends a pmap event to a Google Cloud PubSub topic.
func (p *PubSubMessenger) Send(ctx context.Context, event *v1alpha1.PmapEvent) error {
	eventBytes, err := protojson.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event json: %w", err)
	}

	result := p.topic.Publish(ctx, &pubsub.Message{
		Data: eventBytes,
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
