// Code generated by protoc-gen-go. DO NOT EDIT.
// source: meta.proto

/*
Package pb is a generated protocol buffer package.

It is generated from these files:
	meta.proto

It has these top-level messages:
	MapperMetadata
*/
package pb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type MapperMetadata struct {
	HighestFullyCompletedSeq int64 `protobuf:"varint,1,opt,name=highest_fully_completed_seq,json=highestFullyCompletedSeq" json:"highest_fully_completed_seq,omitempty"`
}

func (m *MapperMetadata) Reset()                    { *m = MapperMetadata{} }
func (m *MapperMetadata) String() string            { return proto.CompactTextString(m) }
func (*MapperMetadata) ProtoMessage()               {}
func (*MapperMetadata) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *MapperMetadata) GetHighestFullyCompletedSeq() int64 {
	if m != nil {
		return m.HighestFullyCompletedSeq
	}
	return 0
}

func init() {
	proto.RegisterType((*MapperMetadata)(nil), "pb.MapperMetadata")
}

func init() { proto.RegisterFile("meta.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 113 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0xca, 0x4d, 0x2d, 0x49,
	0xd4, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x2a, 0x48, 0x52, 0xf2, 0xe7, 0xe2, 0xf3, 0x4d,
	0x2c, 0x28, 0x48, 0x2d, 0xf2, 0x4d, 0x2d, 0x49, 0x4c, 0x49, 0x2c, 0x49, 0x14, 0xb2, 0xe5, 0x92,
	0xce, 0xc8, 0x4c, 0xcf, 0x48, 0x2d, 0x2e, 0x89, 0x4f, 0x2b, 0xcd, 0xc9, 0xa9, 0x8c, 0x4f, 0xce,
	0xcf, 0x2d, 0xc8, 0x49, 0x2d, 0x49, 0x4d, 0x89, 0x2f, 0x4e, 0x2d, 0x94, 0x60, 0x54, 0x60, 0xd4,
	0x60, 0x0e, 0x92, 0x80, 0x2a, 0x71, 0x03, 0xa9, 0x70, 0x86, 0x29, 0x08, 0x4e, 0x2d, 0x4c, 0x62,
	0x03, 0x9b, 0x6d, 0x0c, 0x08, 0x00, 0x00, 0xff, 0xff, 0xf0, 0x6a, 0x65, 0x6b, 0x69, 0x00, 0x00,
	0x00,
}
