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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.21.12
// source: pmap_event.proto

package v1alpha1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	anypb "google.golang.org/protobuf/types/known/anypb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// A representation of an event associated with creation/modification of privacy related information represented by a payload.
type PmapEvent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Required.
	Payload *anypb.Any `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
	// Required. The type of the payload such as resource mapping and retention plan.
	Type string `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	// Required.
	Timestamp *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// Required. The source of the payload.
	GithubSource *GitHubSource `protobuf:"bytes,4,opt,name=github_source,json=githubSource,proto3" json:"github_source,omitempty"`
}

func (x *PmapEvent) Reset() {
	*x = PmapEvent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pmap_event_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PmapEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PmapEvent) ProtoMessage() {}

func (x *PmapEvent) ProtoReflect() protoreflect.Message {
	mi := &file_pmap_event_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PmapEvent.ProtoReflect.Descriptor instead.
func (*PmapEvent) Descriptor() ([]byte, []int) {
	return file_pmap_event_proto_rawDescGZIP(), []int{0}
}

func (x *PmapEvent) GetPayload() *anypb.Any {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *PmapEvent) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *PmapEvent) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *PmapEvent) GetGithubSource() *GitHubSource {
	if x != nil {
		return x.GithubSource
	}
	return nil
}

type GitHubSource struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Required. The repository name where the payload is located.
	RepoName string `protobuf:"bytes,1,opt,name=repo_name,json=repoName,proto3" json:"repo_name,omitempty"`
	// Required. The file path of the payload.
	FilePath string `protobuf:"bytes,2,opt,name=file_path,json=filePath,proto3" json:"file_path,omitempty"`
	// Required. The git commit.
	Commit string `protobuf:"bytes,3,opt,name=commit,proto3" json:"commit,omitempty"`
	// Required. The github workflow that triggered the pmap event.
	// Example: pmap-snapshot-file-change
	Workflow string `protobuf:"bytes,4,opt,name=workflow,proto3" json:"workflow,omitempty"`
	// Required. The sha for the github workflow.
	// Example: 6a558007186d9a4ceb17590166a40f173e5df3ff
	WorkflowSha string `protobuf:"bytes,5,opt,name=workflow_sha,json=workflowSha,proto3" json:"workflow_sha,omitempty"`
	// Required. The timestamp when workflow is triggered.
	// Example: 2023-04-25T17:44:57Z
	WorkflowTriggeredTimestamp *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=workflow_triggered_timestamp,json=workflowTriggeredTimestamp,proto3" json:"workflow_triggered_timestamp,omitempty"`
	// Required. The workflow run id.
	// Example: 5050509831
	WorkflowRunId string `protobuf:"bytes,7,opt,name=workflow_run_id,json=workflowRunId,proto3" json:"workflow_run_id,omitempty"`
	// Required. The workflow run attempts.
	// Example: 1
	WorkflowRunAttempt int64 `protobuf:"varint,8,opt,name=workflow_run_attempt,json=workflowRunAttempt,proto3" json:"workflow_run_attempt,omitempty"`
}

func (x *GitHubSource) Reset() {
	*x = GitHubSource{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pmap_event_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GitHubSource) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GitHubSource) ProtoMessage() {}

func (x *GitHubSource) ProtoReflect() protoreflect.Message {
	mi := &file_pmap_event_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GitHubSource.ProtoReflect.Descriptor instead.
func (*GitHubSource) Descriptor() ([]byte, []int) {
	return file_pmap_event_proto_rawDescGZIP(), []int{1}
}

func (x *GitHubSource) GetRepoName() string {
	if x != nil {
		return x.RepoName
	}
	return ""
}

func (x *GitHubSource) GetFilePath() string {
	if x != nil {
		return x.FilePath
	}
	return ""
}

func (x *GitHubSource) GetCommit() string {
	if x != nil {
		return x.Commit
	}
	return ""
}

func (x *GitHubSource) GetWorkflow() string {
	if x != nil {
		return x.Workflow
	}
	return ""
}

func (x *GitHubSource) GetWorkflowSha() string {
	if x != nil {
		return x.WorkflowSha
	}
	return ""
}

func (x *GitHubSource) GetWorkflowTriggeredTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.WorkflowTriggeredTimestamp
	}
	return nil
}

func (x *GitHubSource) GetWorkflowRunId() string {
	if x != nil {
		return x.WorkflowRunId
	}
	return ""
}

func (x *GitHubSource) GetWorkflowRunAttempt() int64 {
	if x != nil {
		return x.WorkflowRunAttempt
	}
	return 0
}

var File_pmap_event_proto protoreflect.FileDescriptor

var file_pmap_event_proto_rawDesc = []byte{
	0x0a, 0x10, 0x70, 0x6d, 0x61, 0x70, 0x5f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x0b, 0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2e, 0x70, 0x6d, 0x61, 0x70, 0x1a,
	0x19, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x61, 0x6e, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xc9, 0x01, 0x0a, 0x09,
	0x50, 0x6d, 0x61, 0x70, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x2e, 0x0a, 0x07, 0x70, 0x61, 0x79,
	0x6c, 0x6f, 0x61, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x41, 0x6e, 0x79,
	0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x38, 0x0a,
	0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x3e, 0x0a, 0x0d, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x5f, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19,
	0x2e, 0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2e, 0x70, 0x6d, 0x61, 0x70, 0x2e, 0x47, 0x69, 0x74,
	0x48, 0x75, 0x62, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x52, 0x0c, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x22, 0xd7, 0x02, 0x0a, 0x0c, 0x47, 0x69, 0x74, 0x48,
	0x75, 0x62, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x72, 0x65, 0x70, 0x6f,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x72, 0x65, 0x70,
	0x6f, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x70, 0x61,
	0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x50, 0x61,
	0x74, 0x68, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x77, 0x6f,
	0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x77, 0x6f,
	0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x12, 0x21, 0x0a, 0x0c, 0x77, 0x6f, 0x72, 0x6b, 0x66, 0x6c,
	0x6f, 0x77, 0x5f, 0x73, 0x68, 0x61, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x77, 0x6f,
	0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x53, 0x68, 0x61, 0x12, 0x5c, 0x0a, 0x1c, 0x77, 0x6f, 0x72,
	0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x5f, 0x74, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x65, 0x64, 0x5f,
	0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x1a, 0x77, 0x6f, 0x72,
	0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x54, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x65, 0x64, 0x54, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x26, 0x0a, 0x0f, 0x77, 0x6f, 0x72, 0x6b, 0x66,
	0x6c, 0x6f, 0x77, 0x5f, 0x72, 0x75, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0d, 0x77, 0x6f, 0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x52, 0x75, 0x6e, 0x49, 0x64, 0x12,
	0x30, 0x0a, 0x14, 0x77, 0x6f, 0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x5f, 0x72, 0x75, 0x6e, 0x5f,
	0x61, 0x74, 0x74, 0x65, 0x6d, 0x70, 0x74, 0x18, 0x08, 0x20, 0x01, 0x28, 0x03, 0x52, 0x12, 0x77,
	0x6f, 0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x52, 0x75, 0x6e, 0x41, 0x74, 0x74, 0x65, 0x6d, 0x70,
	0x74, 0x42, 0x26, 0x5a, 0x24, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2f, 0x70, 0x6d, 0x61, 0x70, 0x2f, 0x61, 0x70, 0x69, 0x73,
	0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_pmap_event_proto_rawDescOnce sync.Once
	file_pmap_event_proto_rawDescData = file_pmap_event_proto_rawDesc
)

func file_pmap_event_proto_rawDescGZIP() []byte {
	file_pmap_event_proto_rawDescOnce.Do(func() {
		file_pmap_event_proto_rawDescData = protoimpl.X.CompressGZIP(file_pmap_event_proto_rawDescData)
	})
	return file_pmap_event_proto_rawDescData
}

var file_pmap_event_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_pmap_event_proto_goTypes = []interface{}{
	(*PmapEvent)(nil),             // 0: abcxyz.pmap.PmapEvent
	(*GitHubSource)(nil),          // 1: abcxyz.pmap.GitHubSource
	(*anypb.Any)(nil),             // 2: google.protobuf.Any
	(*timestamppb.Timestamp)(nil), // 3: google.protobuf.Timestamp
}
var file_pmap_event_proto_depIdxs = []int32{
	2, // 0: abcxyz.pmap.PmapEvent.payload:type_name -> google.protobuf.Any
	3, // 1: abcxyz.pmap.PmapEvent.timestamp:type_name -> google.protobuf.Timestamp
	1, // 2: abcxyz.pmap.PmapEvent.github_source:type_name -> abcxyz.pmap.GitHubSource
	3, // 3: abcxyz.pmap.GitHubSource.workflow_triggered_timestamp:type_name -> google.protobuf.Timestamp
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_pmap_event_proto_init() }
func file_pmap_event_proto_init() {
	if File_pmap_event_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pmap_event_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PmapEvent); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_pmap_event_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GitHubSource); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_pmap_event_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pmap_event_proto_goTypes,
		DependencyIndexes: file_pmap_event_proto_depIdxs,
		MessageInfos:      file_pmap_event_proto_msgTypes,
	}.Build()
	File_pmap_event_proto = out.File
	file_pmap_event_proto_rawDesc = nil
	file_pmap_event_proto_goTypes = nil
	file_pmap_event_proto_depIdxs = nil
}
