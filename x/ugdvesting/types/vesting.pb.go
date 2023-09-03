// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.12.4
// source: proto/ugdvesting/ugdvesting/vesting.proto

package types

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

type VestingData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Address   string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Amount    int64  `protobuf:"varint,2,opt,name=amount,proto3" json:"amount,omitempty"`
	Start     int64  `protobuf:"varint,3,opt,name=start,proto3" json:"start,omitempty"`       // Use timestamp type if you want to store it as a timestamp
	Duration  int64  `protobuf:"varint,4,opt,name=duration,proto3" json:"duration,omitempty"` // Duration in seconds
	Parts     int32  `protobuf:"varint,5,opt,name=parts,proto3" json:"parts,omitempty"`
	Block     int64  `protobuf:"varint,6,opt,name=block,proto3" json:"block,omitempty"`
	Percent   int32  `protobuf:"varint,7,opt,name=percent,proto3" json:"percent,omitempty"`
	Processed bool   `protobuf:"varint,8,opt,name=processed,proto3" json:"processed,omitempty"`
}

func (x *VestingData) Reset() {
	*x = VestingData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_ugdvesting_ugdvesting_vesting_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VestingData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VestingData) ProtoMessage() {}

func (x *VestingData) ProtoReflect() protoreflect.Message {
	mi := &file_proto_ugdvesting_ugdvesting_vesting_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VestingData.ProtoReflect.Descriptor instead.
func (*VestingData) Descriptor() ([]byte, []int) {
	return file_proto_ugdvesting_ugdvesting_vesting_proto_rawDescGZIP(), []int{0}
}

func (x *VestingData) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *VestingData) GetAmount() int64 {
	if x != nil {
		return x.Amount
	}
	return 0
}

func (x *VestingData) GetStart() int64 {
	if x != nil {
		return x.Start
	}
	return 0
}

func (x *VestingData) GetDuration() int64 {
	if x != nil {
		return x.Duration
	}
	return 0
}

func (x *VestingData) GetParts() int32 {
	if x != nil {
		return x.Parts
	}
	return 0
}

func (x *VestingData) GetBlock() int64 {
	if x != nil {
		return x.Block
	}
	return 0
}

func (x *VestingData) GetPercent() int32 {
	if x != nil {
		return x.Percent
	}
	return 0
}

func (x *VestingData) GetProcessed() bool {
	if x != nil {
		return x.Processed
	}
	return false
}

var File_proto_ugdvesting_ugdvesting_vesting_proto protoreflect.FileDescriptor

var file_proto_ugdvesting_ugdvesting_vesting_proto_rawDesc = []byte{
	0x0a, 0x29, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x75, 0x67, 0x64, 0x76, 0x65, 0x73, 0x74, 0x69,
	0x6e, 0x67, 0x2f, 0x75, 0x67, 0x64, 0x76, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67, 0x2f, 0x76, 0x65,
	0x73, 0x74, 0x69, 0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x15, 0x75, 0x67, 0x64,
	0x76, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67, 0x2e, 0x75, 0x67, 0x64, 0x76, 0x65, 0x73, 0x74, 0x69,
	0x6e, 0x67, 0x22, 0xd5, 0x01, 0x0a, 0x0b, 0x56, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67, 0x44, 0x61,
	0x74, 0x61, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x16, 0x0a, 0x06,
	0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x61, 0x6d,
	0x6f, 0x75, 0x6e, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x72, 0x74, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x05, 0x73, 0x74, 0x61, 0x72, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x64, 0x75,
	0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x64, 0x75,
	0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x14, 0x0a, 0x05, 0x70, 0x61, 0x72, 0x74, 0x73, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x70, 0x61, 0x72, 0x74, 0x73, 0x12, 0x14, 0x0a, 0x05,
	0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x62, 0x6c, 0x6f,
	0x63, 0x6b, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x07, 0x70, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74, 0x12, 0x1c, 0x0a, 0x09,
	0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64, 0x18, 0x08, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x09, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64, 0x42, 0x53, 0x5a, 0x51, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x75, 0x6e, 0x69, 0x67, 0x72, 0x69, 0x64,
	0x2d, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x2f, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2d,
	0x73, 0x64, 0x6b, 0x2d, 0x75, 0x6e, 0x69, 0x67, 0x72, 0x69, 0x64, 0x2d, 0x68, 0x65, 0x64, 0x67,
	0x65, 0x68, 0x6f, 0x67, 0x2d, 0x76, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67, 0x2f, 0x78, 0x2f, 0x75,
	0x67, 0x64, 0x76, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_ugdvesting_ugdvesting_vesting_proto_rawDescOnce sync.Once
	file_proto_ugdvesting_ugdvesting_vesting_proto_rawDescData = file_proto_ugdvesting_ugdvesting_vesting_proto_rawDesc
)

func file_proto_ugdvesting_ugdvesting_vesting_proto_rawDescGZIP() []byte {
	file_proto_ugdvesting_ugdvesting_vesting_proto_rawDescOnce.Do(func() {
		file_proto_ugdvesting_ugdvesting_vesting_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_ugdvesting_ugdvesting_vesting_proto_rawDescData)
	})
	return file_proto_ugdvesting_ugdvesting_vesting_proto_rawDescData
}

var file_proto_ugdvesting_ugdvesting_vesting_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_proto_ugdvesting_ugdvesting_vesting_proto_goTypes = []interface{}{
	(*VestingData)(nil), // 0: ugdvesting.ugdvesting.VestingData
}
var file_proto_ugdvesting_ugdvesting_vesting_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_proto_ugdvesting_ugdvesting_vesting_proto_init() }
func file_proto_ugdvesting_ugdvesting_vesting_proto_init() {
	if File_proto_ugdvesting_ugdvesting_vesting_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_ugdvesting_ugdvesting_vesting_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VestingData); i {
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
			RawDescriptor: file_proto_ugdvesting_ugdvesting_vesting_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proto_ugdvesting_ugdvesting_vesting_proto_goTypes,
		DependencyIndexes: file_proto_ugdvesting_ugdvesting_vesting_proto_depIdxs,
		MessageInfos:      file_proto_ugdvesting_ugdvesting_vesting_proto_msgTypes,
	}.Build()
	File_proto_ugdvesting_ugdvesting_vesting_proto = out.File
	file_proto_ugdvesting_ugdvesting_vesting_proto_rawDesc = nil
	file_proto_ugdvesting_ugdvesting_vesting_proto_goTypes = nil
	file_proto_ugdvesting_ugdvesting_vesting_proto_depIdxs = nil
}
