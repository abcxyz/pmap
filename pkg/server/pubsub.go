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

const (
	MaxTopicDataBytes      = 10_000_000
	MaxTopicAttrValueBytes = 1024
)

// PubSubMessenger implements the Messenger interface for Google Cloud PubSub.
type PubSubMessenger struct {
	topic *pubsub.Topic
}

// NewPubSubMessenger creates a new instance of the PubSubMessenger.
func NewPubSubMessenger(topic *pubsub.Topic) *PubSubMessenger {
	return &PubSubMessenger{topic: topic}
}

func (p *PubSubMessenger) Send(ctx context.Context, data []byte, attr map[string]string) error {
	m, err := limitedSizeMessage(data, attr)
	if err != nil {
		return fmt.Errorf("pubsub failed to publish message: %w", err)
	}

	fmt.Println(p.topic)
	result := p.topic.Publish(ctx, m)

	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("pubsub failed to get result returned from publish : %w", err)
	}
	return nil
}

func limitedSizeMessage(data []byte, attr map[string]string) (*pubsub.Message, error) {
	if len(data) > MaxTopicDataBytes {
		return nil, fmt.Errorf("data length(%d) exceed max size allowed(%d)", len(data), MaxTopicDataBytes)
	}
	d := data[:min(MaxTopicDataBytes, len(data))]

	for key, value := range attr {
		v := []byte(value)[:min(MaxTopicAttrValueBytes, len(value))]
		attr[key] = string(v)
	}

	return &pubsub.Message{
		Data:       d,
		Attributes: attr,
	}, nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
