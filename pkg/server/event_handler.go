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

// Package server is the base server for the pmap event ingestion.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/protoutil"
	"google.golang.org/protobuf/proto"
)

const (
	httpRequestSizeLimitInBytes = 256_000
	gcsObjectSizeLimitInBytes   = 25_000_000
)

// EventHandler retrieves GCS objects upon receiving GCS notifications
// via Pub/Sub, calls a list of processors to process the objects, and
// lastly passes the objects downstream.
//
// The GCS object could be any proto message type. But an instance of
// Handler can only handle one type of proto message.
//
// TODO(#19): passes the objects downstream.
type EventHandler[T any, P ProtoWrapper[T]] struct {
	client     *storage.Client
	processors []Processor[P]
}

type HandlerOpts struct {
	client *storage.Client
}

// Define your option to change HandlerOpts
type Option func(context.Context, *HandlerOpts) (*HandlerOpts, error)

func WithStorageClient(client *storage.Client) Option {
	return func(_ context.Context, opts *HandlerOpts) (*HandlerOpts, error) {
		opts.client = client
		return opts, nil
	}
}

// Create a new Handler with the given processors and handler options.
//
//	// Assume you have processor to handle structpb.Struct.
//	type MyProcessor struct {}
//	func (p *MyProcessor) Process(context.Context, *structpb.Struct) error { return nil }
//	// You can create a handler for that type of processors.
//	h := NewHandler(ctx, []Processor[*structpb.Struct]{&MyProcessor{}})
func NewHandler[T any, P ProtoWrapper[T]](ctx context.Context, ps []Processor[P], opts ...Option) (*EventHandler[T, P], error) {
	h := &EventHandler[T, P]{
		processors: ps,
	}
	handlerOpt := &HandlerOpts{}
	for _, opt := range opts {
		_, err := opt(ctx, handlerOpt)
		if err != nil {
			return nil, fmt.Errorf("failed to apply handler options: %w", err)
		}
	}
	if handlerOpt != nil {
		h.client = handlerOpt.client
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
// This helps to use generics to initialize proto messages without knowing their types.
type ProtoWrapper[T any] interface {
	proto.Message
	*T
}

// A generic interface for processing proto messages.
// Any type that processes proto can implement this interface.
//
// For example, someProcessor implements
// Process(context.Context, *structpb.Struct) is of type
// Processor[*structpb.Struct].
type Processor[P proto.Message] interface {
	Process(context.Context, P) error
}

// PubSubMessage is the payload of a [Pub/Sub message].
//
// [Pub/Sub message]: https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
type PubSubMessage struct {
	Message struct {
		Data       []byte            `json:"data,omitempty"`
		Attributes map[string]string `json:"attributes"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

// HTTPHandler provides an [http.Handler] that accepts [GCS notifications]
// in HTTP requests and calls [Handle] to handle the events.
//
// [http.Handler]: https://pkg.go.dev/net/http#Handler
// [GCS notifications]: https://cloud.google.com/storage/docs/pubsub-notifications#format
// [Handle]: https://pkg.go.dev/github.com/abcxyz/pmap/pkg/server#EventHandler.Handle
func (h *EventHandler[T, P]) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.FromContext(ctx)

		// Handle Pub/Sub http request which is a GCS notification message.
		body, err := io.ReadAll(io.LimitReader(r.Body, httpRequestSizeLimitInBytes))
		if err != nil {
			logger.Errorw("failed to read the request body", "code", http.StatusBadRequest, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Convert the GCS notification message into a PubSubMessage.
		var m PubSubMessage
		// Handle message body(base64-encoded) decoding.
		if err := json.Unmarshal(body, &m); err != nil {
			logger.Errorw("failed to unmarshal the request body", "code", http.StatusBadRequest, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		logger.Debug("%T: handling message from Pub/Sub subscription: %q", h, m.Subscription)

		if err := h.Handle(ctx, m); err != nil {
			logger.Errorw("failed to handle request", "code", http.StatusInternalServerError, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, "OK")
	})
}

// Handle retrieves a GCS object with the given [GCS notification],
// processes the object with the list of processors, and passes it downstream.
//
// [GCS notification]: https://cloud.google.com/storage/docs/pubsub-notifications#format
func (h *EventHandler[T, P]) Handle(ctx context.Context, m PubSubMessage) error {
	// Get the GCS object as a proto message given GCS notification information.
	p, err := h.getGCSObjectProto(ctx, m.Message.Attributes)
	if err != nil {
		return fmt.Errorf("failed to get GCS object: %w", err)
	}

	for _, processor := range h.processors {
		if err := processor.Process(ctx, p); err != nil {
			return fmt.Errorf("failed to process object: %w", err)
		}
	}

	// TODO(#19): Create pmap event and pass it downstream.
	// TODO(#21): Add additional metadata to pmap event.
	// TODO(#20): we need to have a way to differentiate retryable err vs. not.
	// For non-retryable error, we need to have them enter a different BQ table per design.
	return nil
}

// getGCSObjectProto calls the GCS storage client with objAttrs information, and returns the object as a proto message.
func (h *EventHandler[T, P]) getGCSObjectProto(ctx context.Context, objAttrs map[string]string) (P, error) {
	// Get bucket and object id from message attributes.
	bucketID, found := objAttrs["bucketId"]
	if !found {
		return nil, fmt.Errorf("bucket ID not found")
	}
	objectID, found := objAttrs["objectId"]
	if !found {
		return nil, fmt.Errorf("object ID not found")
	}

	// Read the object from bucket.
	rc, err := h.client.Bucket(bucketID).Object(objectID).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS object reader: %w", err)
	}
	defer rc.Close()
	yb, err := io.ReadAll(io.LimitReader(rc, gcsObjectSizeLimitInBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read object from GCS: %w", err)
	}

	// Unmarshal the object yaml bytes into a proto message wrapper.
	p := P(new(T))
	if err := protoutil.UnmarshalYAML(yb, p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal object yaml: %w", err)
	}
	return p, nil
}
