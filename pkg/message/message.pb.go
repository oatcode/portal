// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v5.29.3
// source: message.proto

package message

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Message_Type int32

const (
	Message_PROXY_CONNECT       Message_Type = 0
	Message_PROXY_CONNECTED     Message_Type = 1
	Message_SERVICE_UNAVAILABLE Message_Type = 2
	Message_DISCONNECTED        Message_Type = 3
	Message_DATA                Message_Type = 4
	Message_DIRECT_CONNECT      Message_Type = 5
	Message_DIRECT_CONNECTED    Message_Type = 6
)

// Enum value maps for Message_Type.
var (
	Message_Type_name = map[int32]string{
		0: "PROXY_CONNECT",
		1: "PROXY_CONNECTED",
		2: "SERVICE_UNAVAILABLE",
		3: "DISCONNECTED",
		4: "DATA",
		5: "DIRECT_CONNECT",
		6: "DIRECT_CONNECTED",
	}
	Message_Type_value = map[string]int32{
		"PROXY_CONNECT":       0,
		"PROXY_CONNECTED":     1,
		"SERVICE_UNAVAILABLE": 2,
		"DISCONNECTED":        3,
		"DATA":                4,
		"DIRECT_CONNECT":      5,
		"DIRECT_CONNECTED":    6,
	}
)

func (x Message_Type) Enum() *Message_Type {
	p := new(Message_Type)
	*p = x
	return p
}

func (x Message_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Message_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_message_proto_enumTypes[0].Descriptor()
}

func (Message_Type) Type() protoreflect.EnumType {
	return &file_message_proto_enumTypes[0]
}

func (x Message_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Message_Type.Descriptor instead.
func (Message_Type) EnumDescriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{0, 0}
}

type Message_Origin int32

const (
	Message_ORIGIN_LOCAL  Message_Origin = 0
	Message_ORIGIN_REMOTE Message_Origin = 1
)

// Enum value maps for Message_Origin.
var (
	Message_Origin_name = map[int32]string{
		0: "ORIGIN_LOCAL",
		1: "ORIGIN_REMOTE",
	}
	Message_Origin_value = map[string]int32{
		"ORIGIN_LOCAL":  0,
		"ORIGIN_REMOTE": 1,
	}
)

func (x Message_Origin) Enum() *Message_Origin {
	p := new(Message_Origin)
	*p = x
	return p
}

func (x Message_Origin) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Message_Origin) Descriptor() protoreflect.EnumDescriptor {
	return file_message_proto_enumTypes[1].Descriptor()
}

func (Message_Origin) Type() protoreflect.EnumType {
	return &file_message_proto_enumTypes[1]
}

func (x Message_Origin) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Message_Origin.Descriptor instead.
func (Message_Origin) EnumDescriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{0, 1}
}

type Message struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Type          Message_Type           `protobuf:"varint,1,opt,name=type,proto3,enum=message.Message_Type" json:"type,omitempty"`
	Origin        Message_Origin         `protobuf:"varint,2,opt,name=origin,proto3,enum=message.Message_Origin" json:"origin,omitempty"`
	Id            uint32                 `protobuf:"varint,3,opt,name=id,proto3" json:"id,omitempty"`
	Address       string                 `protobuf:"bytes,4,opt,name=address,proto3" json:"address,omitempty"`
	Data          []byte                 `protobuf:"bytes,5,opt,name=data,proto3" json:"data,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Message) Reset() {
	*x = Message{}
	mi := &file_message_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Message) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Message) ProtoMessage() {}

func (x *Message) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Message.ProtoReflect.Descriptor instead.
func (*Message) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{0}
}

func (x *Message) GetType() Message_Type {
	if x != nil {
		return x.Type
	}
	return Message_PROXY_CONNECT
}

func (x *Message) GetOrigin() Message_Origin {
	if x != nil {
		return x.Origin
	}
	return Message_ORIGIN_LOCAL
}

func (x *Message) GetId() uint32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Message) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *Message) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_message_proto protoreflect.FileDescriptor

var file_message_proto_rawDesc = string([]byte{
	0x0a, 0x0d, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0xe2, 0x02, 0x0a, 0x07, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x12, 0x29, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x15, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12,
	0x2f, 0x0a, 0x06, 0x6f, 0x72, 0x69, 0x67, 0x69, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x17, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x2e, 0x4f, 0x72, 0x69, 0x67, 0x69, 0x6e, 0x52, 0x06, 0x6f, 0x72, 0x69, 0x67, 0x69, 0x6e,
	0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x02, 0x69, 0x64,
	0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61,
	0x74, 0x61, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x8d,
	0x01, 0x0a, 0x04, 0x54, 0x79, 0x70, 0x65, 0x12, 0x11, 0x0a, 0x0d, 0x50, 0x52, 0x4f, 0x58, 0x59,
	0x5f, 0x43, 0x4f, 0x4e, 0x4e, 0x45, 0x43, 0x54, 0x10, 0x00, 0x12, 0x13, 0x0a, 0x0f, 0x50, 0x52,
	0x4f, 0x58, 0x59, 0x5f, 0x43, 0x4f, 0x4e, 0x4e, 0x45, 0x43, 0x54, 0x45, 0x44, 0x10, 0x01, 0x12,
	0x17, 0x0a, 0x13, 0x53, 0x45, 0x52, 0x56, 0x49, 0x43, 0x45, 0x5f, 0x55, 0x4e, 0x41, 0x56, 0x41,
	0x49, 0x4c, 0x41, 0x42, 0x4c, 0x45, 0x10, 0x02, 0x12, 0x10, 0x0a, 0x0c, 0x44, 0x49, 0x53, 0x43,
	0x4f, 0x4e, 0x4e, 0x45, 0x43, 0x54, 0x45, 0x44, 0x10, 0x03, 0x12, 0x08, 0x0a, 0x04, 0x44, 0x41,
	0x54, 0x41, 0x10, 0x04, 0x12, 0x12, 0x0a, 0x0e, 0x44, 0x49, 0x52, 0x45, 0x43, 0x54, 0x5f, 0x43,
	0x4f, 0x4e, 0x4e, 0x45, 0x43, 0x54, 0x10, 0x05, 0x12, 0x14, 0x0a, 0x10, 0x44, 0x49, 0x52, 0x45,
	0x43, 0x54, 0x5f, 0x43, 0x4f, 0x4e, 0x4e, 0x45, 0x43, 0x54, 0x45, 0x44, 0x10, 0x06, 0x22, 0x2d,
	0x0a, 0x06, 0x4f, 0x72, 0x69, 0x67, 0x69, 0x6e, 0x12, 0x10, 0x0a, 0x0c, 0x4f, 0x52, 0x49, 0x47,
	0x49, 0x4e, 0x5f, 0x4c, 0x4f, 0x43, 0x41, 0x4c, 0x10, 0x00, 0x12, 0x11, 0x0a, 0x0d, 0x4f, 0x52,
	0x49, 0x47, 0x49, 0x4e, 0x5f, 0x52, 0x45, 0x4d, 0x4f, 0x54, 0x45, 0x10, 0x01, 0x42, 0x0d, 0x5a,
	0x0b, 0x70, 0x6b, 0x67, 0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_message_proto_rawDescOnce sync.Once
	file_message_proto_rawDescData []byte
)

func file_message_proto_rawDescGZIP() []byte {
	file_message_proto_rawDescOnce.Do(func() {
		file_message_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_message_proto_rawDesc), len(file_message_proto_rawDesc)))
	})
	return file_message_proto_rawDescData
}

var file_message_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_message_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_message_proto_goTypes = []any{
	(Message_Type)(0),   // 0: message.Message.Type
	(Message_Origin)(0), // 1: message.Message.Origin
	(*Message)(nil),     // 2: message.Message
}
var file_message_proto_depIdxs = []int32{
	0, // 0: message.Message.type:type_name -> message.Message.Type
	1, // 1: message.Message.origin:type_name -> message.Message.Origin
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_message_proto_init() }
func file_message_proto_init() {
	if File_message_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_message_proto_rawDesc), len(file_message_proto_rawDesc)),
			NumEnums:      2,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_message_proto_goTypes,
		DependencyIndexes: file_message_proto_depIdxs,
		EnumInfos:         file_message_proto_enumTypes,
		MessageInfos:      file_message_proto_msgTypes,
	}.Build()
	File_message_proto = out.File
	file_message_proto_goTypes = nil
	file_message_proto_depIdxs = nil
}
