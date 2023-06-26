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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/protoutil"
	"github.com/abcxyz/pmap/apis/v1alpha1"
	"github.com/abcxyz/pmap/pkg/pmaperrors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	httpRequestSizeLimitInBytes = 256_000
	gcsObjectSizeLimitInBytes   = 25_000_000
)

const (
	// AttrKeyProcessErr is the attribute key for process error.
	AttrKeyProcessErr = "ProcessErr"
)

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

// StoppableProcessor is the interface to processors that are stoppable.
type StoppableProcessor[P proto.Message] interface {
	Stop() error
}

// These are metadatas for GCS objects that were uploaded.
// These customs keys are defined in snapshot-file-change
// and snapshot-file-copy workflow.
// https://github.com/abcxyz/pmap/blob/main/.github/workflows/snapshot-file-change.yml#L74-L78
const (
	MetadataKeyGitHubCommit               = "github-commit"
	MetadataKeyGitHubRepo                 = "github-repo"
	MetadataKeyWorkflow                   = "github-workflow"
	MetadataKeyWorkflowSha                = "github-workflow-sha"
	MetadataKeyWorkflowTriggeredTimestamp = "github-workflow-triggered-timestamp"
	MetadataKeyWorkflowRunID              = "github-run-id"
	MetadataKeyWorkflowRunAttempt         = "github-run-attempt"
	GCSPathSeparatorKey                   = "/gh-prefix/"
)

// An interface for sending pmap event downstream.
type Messenger interface {
	Send(context.Context, []byte, map[string]string) error
}

// EventHandler retrieves GCS objects upon receiving GCS notifications
// via Pub/Sub, calls a list of processors to process the objects, and
// lastly passes the objects downstream. The successMessenger handles
// successfully processed objects, the failureMessenger handles failure
// events.
//
// The GCS object could be any proto message type. But an instance of
// Handler can only handle one type of proto message.
type EventHandler[T any, P ProtoWrapper[T]] struct {
	client           *storage.Client
	processors       []Processor[P]
	successMessenger Messenger
	failureMessenger Messenger
}

// HandlerOpts available when creating an EventHandler such as GCS storage client
// and Messenger for failure events.
type HandlerOpts struct {
	client           *storage.Client
	failureMessenger Messenger
}

// Define your option to change HandlerOpts.
type Option func(context.Context, *HandlerOpts) (*HandlerOpts, error)

// WithStorageClient returns an option to set the GCS storage client when creating
// an EventHandler.
func WithStorageClient(client *storage.Client) Option {
	return func(_ context.Context, opts *HandlerOpts) (*HandlerOpts, error) {
		opts.client = client
		return opts, nil
	}
}

// WithFailureMessenger returns an option to set the Messenger for unsuccessfully
// processed pmap event when creating an EventHandler.
func WithFailureMessenger(msger Messenger) Option {
	return func(_ context.Context, opts *HandlerOpts) (*HandlerOpts, error) {
		opts.failureMessenger = msger
		return opts, nil
	}
}

// Create a new Handler with the given processors, successMessenger, and handler options.
// failureMessenger will default to NoopMessenger if not provided.
//
//	// Assume you have processor to handle structpb.Struct.
//	type MyProcessor struct {}
//	func (p *MyProcessor) Process(context.Context, *structpb.Struct) error { return nil }
//	// You can create a handler for that type of processors.
//	h := NewHandler(ctx, []Processor[*structpb.Struct]{&MyProcessor{}}, msgr, opts...)
func NewHandler[T any, P ProtoWrapper[T]](ctx context.Context, ps []Processor[P], successMessenger Messenger, opts ...Option) (*EventHandler[T, P], error) {
	h := &EventHandler[T, P]{
		processors:       ps,
		successMessenger: successMessenger,
	}
	handlerOpt := &HandlerOpts{}
	for _, opt := range opts {
		_, err := opt(ctx, handlerOpt)
		if err != nil {
			return nil, fmt.Errorf("failed to apply handler options: %w", err)
		}
	}
	h.client = handlerOpt.client
	h.failureMessenger = handlerOpt.failureMessenger

	if h.successMessenger == nil {
		return nil, fmt.Errorf("successMessenger cannot be nil")
	}

	// Default to no-op Messenger.
	if h.failureMessenger == nil {
		h.failureMessenger = &NoopMessenger{}
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

// PubSubMessage is the payload of a [Pub/Sub message].
//
// GCS objects' custom metadata will be included in [Data].
// [Attributes]: includes bucketID and objectID info.
// [Pub/Sub message]: https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
// [Data]: https://cloud.google.com/storage/docs/json_api/v1/objects#resource-representations
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
// Object's metadata change will be included in payload of the notification.
//
// [GCS notifications]: https://cloud.google.com/storage/docs/pubsub-notifications#format
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

		// Extract out notification information.
		n := pubsub.Message{
			Data:       m.Message.Data, // Notification payload.
			Attributes: m.Message.Attributes,
		}
		if err := h.Handle(ctx, n); err != nil {
			logger.Errorw("failed to handle request", "code", http.StatusInternalServerError,
				"error", err, "bucketId", n.Attributes["bucketId"], "objectId", n.Attributes["objectId"])
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
func (h *EventHandler[T, P]) Handle(ctx context.Context, m pubsub.Message) error {
	logger := logging.FromContext(ctx)

	eventBytes, err := h.generatePmapEventBytes(ctx, m)

	attr := map[string]string{}

	if err != nil {
		// We only write the failure event if it's an user facing error.
		if !pmaperrors.Is(err) {
			return err
		}
		attr[AttrKeyProcessErr] = err.Error()
		logger.Errorw(err.Error(), "bucketId", m.Attributes["bucketId"], "objectId", m.Attributes["objectId"])
		if err := h.failureMessenger.Send(ctx, eventBytes, attr); err != nil {
			return fmt.Errorf("failed to send failure event downstream: %w", err)
		}
		return nil
	}
	if err := h.successMessenger.Send(ctx, eventBytes, attr); err != nil {
		return fmt.Errorf("failed to send succuss event downstream: %w", err)
	}
	return nil
}

func (h *EventHandler[T, P]) generatePmapEventBytes(ctx context.Context, m pubsub.Message) ([]byte, error) {
	// Get the GCS object as a proto message given GCS notification information.
	p, err := h.getGCSObjectProto(ctx, m.Attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to get GCS object: %w", err)
	}
	var processErr error
	for _, processor := range h.processors {
		if err := processor.Process(ctx, p); err != nil {
			processErr = fmt.Errorf("failed to process object: %w", err)
			break
		}
	}

	payload, err := anypb.New(p)
	if err != nil {
		return nil, fmt.Errorf("failed to convert object to pmap event payload: %w", err)
	}

	var gr *v1alpha1.GitHubSource
	if m.Attributes["payloadFormat"] == "JSON_API_V1" {
		gr, err = parseGitHubSource(ctx, m.Data, m.Attributes)
		if err != nil {
			// Join with the processErr. We don't want to lose the user facing error if it's not nil.
			return nil, errors.Join(processErr, fmt.Errorf("failed to parse metadata: %w", err))
		}
	}

	event := &v1alpha1.PmapEvent{
		Payload:      payload,
		GithubSource: gr,
	}

	eventBytes, err := protojson.Marshal(event)
	if err != nil {
		// Join with the processErr. We don't want to lose the user facing error if it's not nil.
		return nil, errors.Join(processErr, fmt.Errorf("failed to marshal event to byte: %w", err))
	}
	return eventBytes, processErr
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

	// Convert the object yaml bytes into a proto message wrapper.
	p := P(new(T))
	if err := protoutil.FromYAML(yb, p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal object yaml: %w", err)
	}
	return p, nil
}

type notificationPayload struct {
	Metadata map[string]string `json:"metadata,omitempty"`
}

func parseGitHubSource(ctx context.Context, data []byte, objAttrs map[string]string) (*v1alpha1.GitHubSource, error) {
	var pm *notificationPayload
	if err := json.Unmarshal(data, &pm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payloadMetadata %w", err)
	}

	var r v1alpha1.GitHubSource

	// Set github-commit. Response with error message when github-commit can not be found.
	c, found := pm.Metadata[MetadataKeyGitHubCommit]
	if !found {
		return nil, fmt.Errorf("github-commit not found")
	} else {
		r.Commit = c
	}

	// Set github-repo. Response with error message when github-repo can not be found.
	rn, found := pm.Metadata[MetadataKeyGitHubRepo]
	if !found {
		return nil, fmt.Errorf("github-repo not found")
	} else {
		r.RepoName = rn
	}

	// Set github-workflow. Response with error message when github-workflow can not be found.
	w, found := pm.Metadata[MetadataKeyWorkflow]
	if !found {
		return nil, fmt.Errorf("github-workflow not found")
	} else {
		r.Workflow = w
	}

	// Set github-workflow-sha. Response with error message when github-workflow-sha can not be found.
	ws, found := pm.Metadata[MetadataKeyWorkflowSha]
	if !found {
		return nil, fmt.Errorf("github-workflow-sha not found")
	} else {
		r.WorkflowSha = ws
	}

	ra, found := pm.Metadata[MetadataKeyWorkflowRunAttempt]
	if found {
		if value, err := strconv.ParseInt(ra, 10, 64); err == nil {
			r.WorkflowRunAttempt = value
		}
	}

	ri, found := pm.Metadata[MetadataKeyWorkflowRunID]
	if found {
		r.WorkflowRunId = ri
	}

	if objectID, found := objAttrs["objectId"]; found {
		parts := strings.Split(objectID, "/gh-prefix/")
		if len(parts) == 2 {
			r.FilePath = parts[1]
		}
	}

	if t, found := pm.Metadata[MetadataKeyWorkflowTriggeredTimestamp]; found {
		date, err := time.Parse(time.RFC3339, t)
		if err != nil {
			return nil, fmt.Errorf("failed to parse date %w", err)
		}
		r.WorkflowTriggeredTimestamp = timestamppb.New(date)
	}
	return &r, nil
}

// NoopMessenger is a no-op implementation of Messenger interface.
type NoopMessenger struct{}

func (m *NoopMessenger) Send(_ context.Context, _ []byte, _ map[string]string) error {
	return nil
}
