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

// NewPubSubMessenger creates a new instance of the PubSubMessenger.
func NewPubSubMessenger(client *pubsub.Client, topic *pubsub.Topic) *PubSubMessenger {
	return &PubSubMessenger{client: client, topic: topic}
}

// Send sends a pmap event to a Google Cloud PubSub topic.
func (p *PubSubMessenger) Send(ctx context.Context, data []byte, attr map[string]string) error {
	result := p.topic.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attr,
	})

	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("pubsub: failed to get result returned from publish : %w", err)
	}
	return nil
}
