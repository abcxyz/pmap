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

	"cloud.google.com/go/storage"
	"google.golang.org/protobuf/proto"
)

// EventHandler retrieves GCS objects upon receiving GCS notifications,
// calls a list of processors to process the objects, and lastly writes
// the objects to BigQuery.

// The GCS object could be any proto message type. But an instance of Handler
// can only handle one type of proto message.
//
// TODO: writes the messages to BigQuery.
type EventHandler[T any, P ProtoMessageWrapper[T]] struct {
	client     *storage.Client
	processors []Processor[P]
	// TODO: Add BigQuery table writer.
}

// Option is the option to set up a EventHandler.
type Option[T any, P ProtoMessageWrapper[T]] func(p *EventHandler[T, P]) (*EventHandler[T, P], error)

// WithClient provides a GCS storage client to the EventHandler.
func WithClient[T any, P ProtoMessageWrapper[T]](client *storage.Client) Option[T, P] {
	return func(p *EventHandler[T, P]) (*EventHandler[T, P], error) {
		p.client = client
		return p, nil
	}
}

// Create a new Handler with the given processors and client options.
func NewHandler[T any, P ProtoMessageWrapper[T]](ctx context.Context, cfg *ServiceConfig, ps []Processor[P], opts ...Option[T, P]) (*EventHandler[T, P], error) {
	h := &EventHandler[T, P]{
		processors: ps,
	}
	for _, opt := range opts {
		var err error
		h, err = opt(h)
		if err != nil {
			return nil, fmt.Errorf("failed to apply client options: %w", err)
		}
	}

	if h.client == nil {
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create the GCS storage client: %w", err)
		}
		h.client = client
	}
	return h, nil
}

// Wrap the proto message interface.
// This is required to handle GCS object without knowing its type.
type ProtoMessageWrapper[T any] interface {
	proto.Message
	*T
}

// A generic interface for processing proto messages.
type Processor[P proto.Message] interface {
	Process(context.Context, P) error
}

// PubSubMessage is the payload of a Pub/Sub event.
// See the documentation for more details:
// https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
type PubSubMessage struct {
	Message struct {
		Attributes map[string]string `json:"attributes"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}
