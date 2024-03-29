// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: pagination/v1/pagination.proto

package paginationv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Arguments struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	First  *int64  `protobuf:"varint,1,opt,name=first,proto3,oneof" json:"first,omitempty"`
	After  *string `protobuf:"bytes,2,opt,name=after,proto3,oneof" json:"after,omitempty"`
	Last   *int64  `protobuf:"varint,3,opt,name=last,proto3,oneof" json:"last,omitempty"`
	Before *string `protobuf:"bytes,4,opt,name=before,proto3,oneof" json:"before,omitempty"`
}

func (x *Arguments) Reset() {
	*x = Arguments{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pagination_v1_pagination_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Arguments) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Arguments) ProtoMessage() {}

func (x *Arguments) ProtoReflect() protoreflect.Message {
	mi := &file_pagination_v1_pagination_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Arguments.ProtoReflect.Descriptor instead.
func (*Arguments) Descriptor() ([]byte, []int) {
	return file_pagination_v1_pagination_proto_rawDescGZIP(), []int{0}
}

func (x *Arguments) GetFirst() int64 {
	if x != nil && x.First != nil {
		return *x.First
	}
	return 0
}

func (x *Arguments) GetAfter() string {
	if x != nil && x.After != nil {
		return *x.After
	}
	return ""
}

func (x *Arguments) GetLast() int64 {
	if x != nil && x.Last != nil {
		return *x.Last
	}
	return 0
}

func (x *Arguments) GetBefore() string {
	if x != nil && x.Before != nil {
		return *x.Before
	}
	return ""
}

type PageInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StartCursor     *string `protobuf:"bytes,1,opt,name=start_cursor,json=startCursor,proto3,oneof" json:"start_cursor,omitempty"`
	EndCursor       *string `protobuf:"bytes,2,opt,name=end_cursor,json=endCursor,proto3,oneof" json:"end_cursor,omitempty"`
	HasNextPage     bool    `protobuf:"varint,3,opt,name=has_next_page,json=hasNextPage,proto3" json:"has_next_page,omitempty"`
	HasPreviousPage bool    `protobuf:"varint,4,opt,name=has_previous_page,json=hasPreviousPage,proto3" json:"has_previous_page,omitempty"`
}

func (x *PageInfo) Reset() {
	*x = PageInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pagination_v1_pagination_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PageInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PageInfo) ProtoMessage() {}

func (x *PageInfo) ProtoReflect() protoreflect.Message {
	mi := &file_pagination_v1_pagination_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PageInfo.ProtoReflect.Descriptor instead.
func (*PageInfo) Descriptor() ([]byte, []int) {
	return file_pagination_v1_pagination_proto_rawDescGZIP(), []int{1}
}

func (x *PageInfo) GetStartCursor() string {
	if x != nil && x.StartCursor != nil {
		return *x.StartCursor
	}
	return ""
}

func (x *PageInfo) GetEndCursor() string {
	if x != nil && x.EndCursor != nil {
		return *x.EndCursor
	}
	return ""
}

func (x *PageInfo) GetHasNextPage() bool {
	if x != nil {
		return x.HasNextPage
	}
	return false
}

func (x *PageInfo) GetHasPreviousPage() bool {
	if x != nil {
		return x.HasPreviousPage
	}
	return false
}

var File_pagination_v1_pagination_proto protoreflect.FileDescriptor

var file_pagination_v1_pagination_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x70, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x76, 0x31, 0x2f,
	0x70, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x0d, 0x70, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x76, 0x31, 0x22,
	0x9f, 0x01, 0x0a, 0x09, 0x41, 0x72, 0x67, 0x75, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x19, 0x0a,
	0x05, 0x66, 0x69, 0x72, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x48, 0x00, 0x52, 0x05,
	0x66, 0x69, 0x72, 0x73, 0x74, 0x88, 0x01, 0x01, 0x12, 0x19, 0x0a, 0x05, 0x61, 0x66, 0x74, 0x65,
	0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x01, 0x52, 0x05, 0x61, 0x66, 0x74, 0x65, 0x72,
	0x88, 0x01, 0x01, 0x12, 0x17, 0x0a, 0x04, 0x6c, 0x61, 0x73, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x03, 0x48, 0x02, 0x52, 0x04, 0x6c, 0x61, 0x73, 0x74, 0x88, 0x01, 0x01, 0x12, 0x1b, 0x0a, 0x06,
	0x62, 0x65, 0x66, 0x6f, 0x72, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x48, 0x03, 0x52, 0x06,
	0x62, 0x65, 0x66, 0x6f, 0x72, 0x65, 0x88, 0x01, 0x01, 0x42, 0x08, 0x0a, 0x06, 0x5f, 0x66, 0x69,
	0x72, 0x73, 0x74, 0x42, 0x08, 0x0a, 0x06, 0x5f, 0x61, 0x66, 0x74, 0x65, 0x72, 0x42, 0x07, 0x0a,
	0x05, 0x5f, 0x6c, 0x61, 0x73, 0x74, 0x42, 0x09, 0x0a, 0x07, 0x5f, 0x62, 0x65, 0x66, 0x6f, 0x72,
	0x65, 0x22, 0xc6, 0x01, 0x0a, 0x08, 0x50, 0x61, 0x67, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x26,
	0x0a, 0x0c, 0x73, 0x74, 0x61, 0x72, 0x74, 0x5f, 0x63, 0x75, 0x72, 0x73, 0x6f, 0x72, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0b, 0x73, 0x74, 0x61, 0x72, 0x74, 0x43, 0x75, 0x72,
	0x73, 0x6f, 0x72, 0x88, 0x01, 0x01, 0x12, 0x22, 0x0a, 0x0a, 0x65, 0x6e, 0x64, 0x5f, 0x63, 0x75,
	0x72, 0x73, 0x6f, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x01, 0x52, 0x09, 0x65, 0x6e,
	0x64, 0x43, 0x75, 0x72, 0x73, 0x6f, 0x72, 0x88, 0x01, 0x01, 0x12, 0x22, 0x0a, 0x0d, 0x68, 0x61,
	0x73, 0x5f, 0x6e, 0x65, 0x78, 0x74, 0x5f, 0x70, 0x61, 0x67, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x0b, 0x68, 0x61, 0x73, 0x4e, 0x65, 0x78, 0x74, 0x50, 0x61, 0x67, 0x65, 0x12, 0x2a,
	0x0a, 0x11, 0x68, 0x61, 0x73, 0x5f, 0x70, 0x72, 0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x5f, 0x70,
	0x61, 0x67, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0f, 0x68, 0x61, 0x73, 0x50, 0x72,
	0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x50, 0x61, 0x67, 0x65, 0x42, 0x0f, 0x0a, 0x0d, 0x5f, 0x73,
	0x74, 0x61, 0x72, 0x74, 0x5f, 0x63, 0x75, 0x72, 0x73, 0x6f, 0x72, 0x42, 0x0d, 0x0a, 0x0b, 0x5f,
	0x65, 0x6e, 0x64, 0x5f, 0x63, 0x75, 0x72, 0x73, 0x6f, 0x72, 0x42, 0x51, 0x5a, 0x4f, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x75, 0x72, 0x70, 0x6f, 0x73, 0x65,
	0x69, 0x6e, 0x70, 0x6c, 0x61, 0x79, 0x2f, 0x67, 0x6f, 0x2d, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e,
	0x73, 0x2f, 0x70, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x70, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x76, 0x31,
	0x3b, 0x70, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x76, 0x31, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pagination_v1_pagination_proto_rawDescOnce sync.Once
	file_pagination_v1_pagination_proto_rawDescData = file_pagination_v1_pagination_proto_rawDesc
)

func file_pagination_v1_pagination_proto_rawDescGZIP() []byte {
	file_pagination_v1_pagination_proto_rawDescOnce.Do(func() {
		file_pagination_v1_pagination_proto_rawDescData = protoimpl.X.CompressGZIP(file_pagination_v1_pagination_proto_rawDescData)
	})
	return file_pagination_v1_pagination_proto_rawDescData
}

var file_pagination_v1_pagination_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_pagination_v1_pagination_proto_goTypes = []interface{}{
	(*Arguments)(nil), // 0: pagination.v1.Arguments
	(*PageInfo)(nil),  // 1: pagination.v1.PageInfo
}
var file_pagination_v1_pagination_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_pagination_v1_pagination_proto_init() }
func file_pagination_v1_pagination_proto_init() {
	if File_pagination_v1_pagination_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pagination_v1_pagination_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Arguments); i {
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
		file_pagination_v1_pagination_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PageInfo); i {
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
	file_pagination_v1_pagination_proto_msgTypes[0].OneofWrappers = []interface{}{}
	file_pagination_v1_pagination_proto_msgTypes[1].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_pagination_v1_pagination_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pagination_v1_pagination_proto_goTypes,
		DependencyIndexes: file_pagination_v1_pagination_proto_depIdxs,
		MessageInfos:      file_pagination_v1_pagination_proto_msgTypes,
	}.Build()
	File_pagination_v1_pagination_proto = out.File
	file_pagination_v1_pagination_proto_rawDesc = nil
	file_pagination_v1_pagination_proto_goTypes = nil
	file_pagination_v1_pagination_proto_depIdxs = nil
}
