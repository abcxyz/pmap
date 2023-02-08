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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
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

// Option is the option to set up a EventHandler.
type Option[T any, P ProtoWrapper[T]] func(p *EventHandler[T, P]) (*EventHandler[T, P], error)

// WithClient provides a GCS storage client to the EventHandler.
func WithClient[T any, P ProtoWrapper[T]](client *storage.Client) Option[T, P] {
	return func(p *EventHandler[T, P]) (*EventHandler[T, P], error) {
		p.client = client
		return p, nil
	}
}

// Create a new Handler with the given processors and handler options.
//
// For example, to create a handler than handles someProto with provided storageClient:
//
//	h := NewHandler(ctx, []Processor[*someProto]{&someProcessor{}, &anotherProcessor{}},WithClient[someProto](storageClient))
func NewHandler[T any, P ProtoWrapper[T]](ctx context.Context, ps []Processor[P], opts ...Option[T, P]) (*EventHandler[T, P], error) {
	h := &EventHandler[T, P]{
		processors: ps,
	}
	for _, opt := range opts {
		var err error
		h, err = opt(h)
		if err != nil {
			return nil, fmt.Errorf("failed to apply handler options: %w", err)
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

// PubSubMessage is the payload of a Pub/Sub event.
// See the documentation for more details:
// https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
type PubSubMessage struct {
	Message struct {
		Attributes map[string]string `json:"attributes"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

// HTTPHandler respondes to http request, and it wraps the Handle function
// with a context and information needed for handling pamp event.
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
		// See GCS notification format here: https://cloud.google.com/storage/docs/pubsub-notifications#format.
		var m PubSubMessage
		// Byte slice unmarshalling handles base64 decoding.
		if err := json.Unmarshal(body, &m); err != nil {
			logger.Errorw("failed to unmarshal the request body", "code", http.StatusBadRequest, "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		logger.Debug("%T: handling message from Pub/Sub subscription: %q", h, m.Subscription)

		err = h.Handle(ctx, m.Message.Attributes)
		if err != nil {
			logger.Errorw("failed to handle request", "code", http.StatusInternalServerError, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, "Ok")
	})
}

// Handle is the core logic of EventHandler, it retrieves GCS object with object information,
// calls a list of processor for processing, and passes it downstream.
func (h *EventHandler[T, P]) Handle(ctx context.Context, objAttrs map[string]string) error {
	// Get the GCS object as a proto message given GCS notification information.
	p, err := h.getGCSObjectProto(ctx, objAttrs)
	if err != nil {
		return fmt.Errorf("failed to get GCS object: %w", err)
	}

	for _, processor := range h.processors {
		if err := processor.Process(ctx, p); err != nil {
			return fmt.Errorf("failed to process object: %w", err)
		}
	}

	// TODO(#19): Create pmap event and pass it downstream...
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
	if err := unmarshalYAML(yb, p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal object yaml: %w", err)
	}

	return p, nil
}

// General func to umarshal yaml bytes to proto.
func unmarshalYAML(b []byte, v proto.Message) error {
	tmp := map[string]any{}
	if err := yaml.Unmarshal(b, tmp); err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}
	jb, err := json.Marshal(tmp)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	if err := protojson.Unmarshal(jb, v); err != nil {
		return fmt.Errorf("failed to unmarshal proto: %w", err)
	}
	return nil
}
