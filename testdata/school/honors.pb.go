// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.12.4
// source: honors.proto

package school

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

type HonorsConfiguration struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Minimum_GPA float64 `protobuf:"fixed64,1,opt,name=Minimum_GPA,json=MinimumGPA,proto3" json:"Minimum_GPA,omitempty"`
}

func (x *HonorsConfiguration) Reset() {
	*x = HonorsConfiguration{}
	if protoimpl.UnsafeEnabled {
		mi := &file_honors_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HonorsConfiguration) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HonorsConfiguration) ProtoMessage() {}

func (x *HonorsConfiguration) ProtoReflect() protoreflect.Message {
	mi := &file_honors_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HonorsConfiguration.ProtoReflect.Descriptor instead.
func (*HonorsConfiguration) Descriptor() ([]byte, []int) {
	return file_honors_proto_rawDescGZIP(), []int{0}
}

func (x *HonorsConfiguration) GetMinimum_GPA() float64 {
	if x != nil {
		return x.Minimum_GPA
	}
	return 0
}

var File_honors_proto protoreflect.FileDescriptor

var file_honors_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x68, 0x6f, 0x6e, 0x6f, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0f,
	0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x73, 0x63, 0x68, 0x6f, 0x6f, 0x6c, 0x22,
	0x36, 0x0a, 0x13, 0x48, 0x6f, 0x6e, 0x6f, 0x72, 0x73, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75,
	0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1f, 0x0a, 0x0b, 0x4d, 0x69, 0x6e, 0x69, 0x6d, 0x75,
	0x6d, 0x5f, 0x47, 0x50, 0x41, 0x18, 0x01, 0x20, 0x01, 0x28, 0x01, 0x52, 0x0a, 0x4d, 0x69, 0x6e,
	0x69, 0x6d, 0x75, 0x6d, 0x47, 0x50, 0x41, 0x42, 0x35, 0x5a, 0x33, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x7a, 0x61, 0x63, 0x68, 0x72, 0x69, 0x73, 0x65, 0x6e,
	0x2f, 0x69, 0x6e, 0x64, 0x69, 0x67, 0x6f, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61,
	0x2f, 0x73, 0x63, 0x68, 0x6f, 0x6f, 0x6c, 0x3b, 0x73, 0x63, 0x68, 0x6f, 0x6f, 0x6c, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_honors_proto_rawDescOnce sync.Once
	file_honors_proto_rawDescData = file_honors_proto_rawDesc
)

func file_honors_proto_rawDescGZIP() []byte {
	file_honors_proto_rawDescOnce.Do(func() {
		file_honors_proto_rawDescData = protoimpl.X.CompressGZIP(file_honors_proto_rawDescData)
	})
	return file_honors_proto_rawDescData
}

var file_honors_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_honors_proto_goTypes = []interface{}{
	(*HonorsConfiguration)(nil), // 0: testdata.school.HonorsConfiguration
}
var file_honors_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_honors_proto_init() }
func file_honors_proto_init() {
	if File_honors_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_honors_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HonorsConfiguration); i {
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
			RawDescriptor: file_honors_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_honors_proto_goTypes,
		DependencyIndexes: file_honors_proto_depIdxs,
		MessageInfos:      file_honors_proto_msgTypes,
	}.Build()
	File_honors_proto = out.File
	file_honors_proto_rawDesc = nil
	file_honors_proto_goTypes = nil
	file_honors_proto_depIdxs = nil
}
