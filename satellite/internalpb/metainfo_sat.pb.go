// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: metainfo_sat.proto

package internalpb

import (
	fmt "fmt"
	math "math"
	time "time"

	proto "github.com/gogo/protobuf/proto"

	pb "common/pb"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = time.Kitchen

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type StreamID struct {
	Bucket               []byte                   `protobuf:"bytes,1,opt,name=bucket,proto3" json:"bucket,omitempty"`
	EncryptedObjectKey   []byte                   `protobuf:"bytes,2,opt,name=encrypted_object_key,json=encryptedObjectKey,proto3" json:"encrypted_object_key,omitempty"`
	Version              int64                    `protobuf:"varint,3,opt,name=version,proto3" json:"version,omitempty"`
	EncryptionParameters *pb.EncryptionParameters `protobuf:"bytes,12,opt,name=encryption_parameters,json=encryptionParameters,proto3" json:"encryption_parameters,omitempty"`
	CreationDate         time.Time                `protobuf:"bytes,5,opt,name=creation_date,json=creationDate,proto3,stdtime" json:"creation_date"`
	ExpirationDate       time.Time                `protobuf:"bytes,6,opt,name=expiration_date,json=expirationDate,proto3,stdtime" json:"expiration_date"`
	MultipartObject      bool                     `protobuf:"varint,11,opt,name=multipart_object,json=multipartObject,proto3" json:"multipart_object,omitempty"`
	SatelliteSignature   []byte                   `protobuf:"bytes,9,opt,name=satellite_signature,json=satelliteSignature,proto3" json:"satellite_signature,omitempty"`
	StreamId             []byte                   `protobuf:"bytes,10,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	Placement            int32                    `protobuf:"varint,13,opt,name=placement,proto3" json:"placement,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                 `json:"-"`
	XXX_unrecognized     []byte                   `json:"-"`
	XXX_sizecache        int32                    `json:"-"`
}

func (m *StreamID) Reset()         { *m = StreamID{} }
func (m *StreamID) String() string { return proto.CompactTextString(m) }
func (*StreamID) ProtoMessage()    {}
func (*StreamID) Descriptor() ([]byte, []int) {
	return fileDescriptor_47c60bd892d94aaf, []int{0}
}
func (m *StreamID) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StreamID.Unmarshal(m, b)
}
func (m *StreamID) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StreamID.Marshal(b, m, deterministic)
}
func (m *StreamID) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StreamID.Merge(m, src)
}
func (m *StreamID) XXX_Size() int {
	return xxx_messageInfo_StreamID.Size(m)
}
func (m *StreamID) XXX_DiscardUnknown() {
	xxx_messageInfo_StreamID.DiscardUnknown(m)
}

var xxx_messageInfo_StreamID proto.InternalMessageInfo

func (m *StreamID) GetBucket() []byte {
	if m != nil {
		return m.Bucket
	}
	return nil
}

func (m *StreamID) GetEncryptedObjectKey() []byte {
	if m != nil {
		return m.EncryptedObjectKey
	}
	return nil
}

func (m *StreamID) GetVersion() int64 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *StreamID) GetEncryptionParameters() *pb.EncryptionParameters {
	if m != nil {
		return m.EncryptionParameters
	}
	return nil
}

func (m *StreamID) GetCreationDate() time.Time {
	if m != nil {
		return m.CreationDate
	}
	return time.Time{}
}

func (m *StreamID) GetExpirationDate() time.Time {
	if m != nil {
		return m.ExpirationDate
	}
	return time.Time{}
}

func (m *StreamID) GetMultipartObject() bool {
	if m != nil {
		return m.MultipartObject
	}
	return false
}

func (m *StreamID) GetSatelliteSignature() []byte {
	if m != nil {
		return m.SatelliteSignature
	}
	return nil
}

func (m *StreamID) GetStreamId() []byte {
	if m != nil {
		return m.StreamId
	}
	return nil
}

func (m *StreamID) GetPlacement() int32 {
	if m != nil {
		return m.Placement
	}
	return 0
}

type SegmentID struct {
	StreamId             *StreamID                 `protobuf:"bytes,1,opt,name=stream_id,json=streamId,proto3" json:"stream_id,omitempty"`
	PartNumber           int32                     `protobuf:"varint,2,opt,name=part_number,json=partNumber,proto3" json:"part_number,omitempty"`
	Index                int32                     `protobuf:"varint,3,opt,name=index,proto3" json:"index,omitempty"`
	RootPieceId          PieceID                   `protobuf:"bytes,5,opt,name=root_piece_id,json=rootPieceId,proto3,customtype=PieceID" json:"root_piece_id"`
	OriginalOrderLimits  []*pb.AddressedOrderLimit `protobuf:"bytes,6,rep,name=original_order_limits,json=originalOrderLimits,proto3" json:"original_order_limits,omitempty"`
	CreationDate         time.Time                 `protobuf:"bytes,7,opt,name=creation_date,json=creationDate,proto3,stdtime" json:"creation_date"`
	SatelliteSignature   []byte                    `protobuf:"bytes,8,opt,name=satellite_signature,json=satelliteSignature,proto3" json:"satellite_signature,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                  `json:"-"`
	XXX_unrecognized     []byte                    `json:"-"`
	XXX_sizecache        int32                     `json:"-"`
}

func (m *SegmentID) Reset()         { *m = SegmentID{} }
func (m *SegmentID) String() string { return proto.CompactTextString(m) }
func (*SegmentID) ProtoMessage()    {}
func (*SegmentID) Descriptor() ([]byte, []int) {
	return fileDescriptor_47c60bd892d94aaf, []int{1}
}
func (m *SegmentID) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SegmentID.Unmarshal(m, b)
}
func (m *SegmentID) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SegmentID.Marshal(b, m, deterministic)
}
func (m *SegmentID) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SegmentID.Merge(m, src)
}
func (m *SegmentID) XXX_Size() int {
	return xxx_messageInfo_SegmentID.Size(m)
}
func (m *SegmentID) XXX_DiscardUnknown() {
	xxx_messageInfo_SegmentID.DiscardUnknown(m)
}

var xxx_messageInfo_SegmentID proto.InternalMessageInfo

func (m *SegmentID) GetStreamId() *StreamID {
	if m != nil {
		return m.StreamId
	}
	return nil
}

func (m *SegmentID) GetPartNumber() int32 {
	if m != nil {
		return m.PartNumber
	}
	return 0
}

func (m *SegmentID) GetIndex() int32 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *SegmentID) GetOriginalOrderLimits() []*pb.AddressedOrderLimit {
	if m != nil {
		return m.OriginalOrderLimits
	}
	return nil
}

func (m *SegmentID) GetCreationDate() time.Time {
	if m != nil {
		return m.CreationDate
	}
	return time.Time{}
}

func (m *SegmentID) GetSatelliteSignature() []byte {
	if m != nil {
		return m.SatelliteSignature
	}
	return nil
}

func init() {
	proto.RegisterType((*StreamID)(nil), "satellite.metainfo.StreamID")
	proto.RegisterType((*SegmentID)(nil), "satellite.metainfo.SegmentID")
}

func init() { proto.RegisterFile("metainfo_sat.proto", fileDescriptor_47c60bd892d94aaf) }

var fileDescriptor_47c60bd892d94aaf = []byte{
	// 548 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x52, 0xcf, 0x6e, 0xd3, 0x4e,
	0x10, 0xfe, 0xf9, 0x17, 0xe5, 0xdf, 0x26, 0x69, 0xaa, 0x6d, 0x8a, 0xac, 0x50, 0x14, 0xab, 0x08,
	0xc9, 0x5c, 0x6c, 0xd4, 0x9e, 0x38, 0x12, 0x85, 0x43, 0xc4, 0x9f, 0x16, 0x07, 0x2e, 0x5c, 0xac,
	0xb5, 0x3d, 0xb5, 0xb6, 0xb5, 0x77, 0xad, 0xdd, 0x09, 0x6a, 0x8e, 0xbc, 0x01, 0x8f, 0xc5, 0x33,
	0x70, 0x28, 0x6f, 0xc1, 0x19, 0x79, 0x1d, 0x3b, 0x91, 0x68, 0x0f, 0x70, 0x9b, 0xf9, 0xe6, 0xdb,
	0x6f, 0x76, 0xbe, 0x19, 0x42, 0x73, 0x40, 0xc6, 0xc5, 0x95, 0x0c, 0x35, 0x43, 0xaf, 0x50, 0x12,
	0x25, 0xa5, 0x9a, 0x21, 0x64, 0x19, 0x47, 0xf0, 0xea, 0xea, 0xf4, 0x10, 0x44, 0xac, 0x36, 0x05,
	0x72, 0x29, 0x2a, 0xd6, 0x94, 0xa4, 0x32, 0x95, 0xdb, 0x78, 0x96, 0x4a, 0x99, 0x66, 0xe0, 0x9b,
	0x2c, 0x5a, 0x5f, 0xf9, 0xc8, 0x73, 0xd0, 0xc8, 0xf2, 0x62, 0x4b, 0x38, 0xa8, 0x85, 0xaa, 0xfc,
	0xf4, 0x57, 0x8b, 0xf4, 0x56, 0xa8, 0x80, 0xe5, 0xcb, 0x05, 0x7d, 0x44, 0x3a, 0xd1, 0x3a, 0xbe,
	0x01, 0xb4, 0x2d, 0xc7, 0x72, 0x87, 0xc1, 0x36, 0xa3, 0x2f, 0xc8, 0x64, 0xdb, 0x15, 0x92, 0x50,
	0x46, 0xd7, 0x10, 0x63, 0x78, 0x03, 0x1b, 0xfb, 0x7f, 0xc3, 0xa2, 0x4d, 0xed, 0xc2, 0x94, 0xde,
	0xc0, 0x86, 0xda, 0xa4, 0xfb, 0x05, 0x94, 0xe6, 0x52, 0xd8, 0x2d, 0xc7, 0x72, 0x5b, 0x41, 0x9d,
	0xd2, 0x4f, 0xe4, 0x78, 0x37, 0x41, 0x58, 0x30, 0xc5, 0x72, 0x40, 0x50, 0xda, 0x1e, 0x3a, 0x96,
	0x3b, 0x38, 0x73, 0xbc, 0xbd, 0xf9, 0x5e, 0x37, 0xe1, 0x65, 0xc3, 0x0b, 0x26, 0x70, 0x0f, 0x4a,
	0x97, 0x64, 0x14, 0x2b, 0x60, 0x46, 0x34, 0x61, 0x08, 0x76, 0xdb, 0xc8, 0x4d, 0xbd, 0xca, 0x10,
	0xaf, 0x36, 0xc4, 0xfb, 0x58, 0x1b, 0x32, 0xef, 0x7d, 0xbf, 0x9b, 0xfd, 0xf7, 0xed, 0xe7, 0xcc,
	0x0a, 0x86, 0xf5, 0xd3, 0x05, 0x43, 0xa0, 0xef, 0xc8, 0x18, 0x6e, 0x0b, 0xae, 0xf6, 0xc4, 0x3a,
	0x7f, 0x21, 0x76, 0xb0, 0x7b, 0x6c, 0xe4, 0x9e, 0x93, 0xc3, 0x7c, 0x9d, 0x21, 0x2f, 0x98, 0xc2,
	0xad, 0x79, 0xf6, 0xc0, 0xb1, 0xdc, 0x5e, 0x30, 0x6e, 0xf0, 0xca, 0x38, 0xea, 0x93, 0xa3, 0x66,
	0xe3, 0xa1, 0xe6, 0xa9, 0x60, 0xb8, 0x56, 0x60, 0xf7, 0x2b, 0x9b, 0x9b, 0xd2, 0xaa, 0xae, 0xd0,
	0xc7, 0xa4, 0xaf, 0xcd, 0xf2, 0x42, 0x9e, 0xd8, 0xc4, 0xd0, 0x7a, 0x15, 0xb0, 0x4c, 0xe8, 0x09,
	0xe9, 0x17, 0x19, 0x8b, 0x21, 0x07, 0x81, 0xf6, 0xc8, 0xb1, 0xdc, 0x76, 0xb0, 0x03, 0x4e, 0xbf,
	0xb6, 0x48, 0x7f, 0x05, 0x69, 0x19, 0x2f, 0x17, 0xf4, 0xe5, 0xbe, 0x90, 0x65, 0xa6, 0x3d, 0xf1,
	0xfe, 0xbc, 0x3e, 0xaf, 0x3e, 0x95, 0xbd, 0x36, 0x33, 0x32, 0x30, 0xa3, 0x89, 0x75, 0x1e, 0x81,
	0x32, 0x37, 0xd1, 0x0e, 0x48, 0x09, 0xbd, 0x37, 0x08, 0x9d, 0x90, 0x36, 0x17, 0x09, 0xdc, 0x9a,
	0x4b, 0x68, 0x07, 0x55, 0x42, 0xcf, 0xc9, 0x48, 0x49, 0x89, 0x61, 0xc1, 0x21, 0x86, 0xb2, 0x6b,
	0xb9, 0xb0, 0xe1, 0x7c, 0x5c, 0xfa, 0xf8, 0xe3, 0x6e, 0xd6, 0xbd, 0x2c, 0xf1, 0xe5, 0x22, 0x18,
	0x94, 0xac, 0x2a, 0x49, 0xe8, 0x07, 0x72, 0x2c, 0x15, 0x4f, 0xb9, 0x60, 0x59, 0x28, 0x55, 0x02,
	0x2a, 0xcc, 0x78, 0xce, 0x51, 0xdb, 0x1d, 0xa7, 0xe5, 0x0e, 0xce, 0x9e, 0xec, 0x3e, 0xfa, 0x2a,
	0x49, 0x14, 0x68, 0x0d, 0xc9, 0x45, 0x49, 0x7b, 0x5b, 0xb2, 0x82, 0xa3, 0xfa, 0xed, 0x0e, 0xbb,
	0xe7, 0x70, 0xba, 0xff, 0x7c, 0x38, 0x0f, 0xac, 0xaf, 0xf7, 0xd0, 0xfa, 0xe6, 0xcf, 0x3e, 0x3f,
	0xd5, 0x28, 0xd5, 0xb5, 0xc7, 0xa5, 0x6f, 0x02, 0xbf, 0x21, 0xf9, 0x5c, 0x20, 0x28, 0xc1, 0xb2,
	0x22, 0x8a, 0x3a, 0xe6, 0x0f, 0xe7, 0xbf, 0x03, 0x00, 0x00, 0xff, 0xff, 0x43, 0x2e, 0x40, 0x6c,
	0x23, 0x04, 0x00, 0x00,
}
