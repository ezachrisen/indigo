// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.12.4
// source: student.proto

package school

import (
	duration "github.com/golang/protobuf/ptypes/duration"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
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

type StudentStatusType int32

const (
	Student_ENROLLED  StudentStatusType = 0
	Student_PROBATION StudentStatusType = 1
)

// Enum value maps for StudentStatusType.
var (
	StudentStatusType_name = map[int32]string{
		0: "ENROLLED",
		1: "PROBATION",
	}
	StudentStatusType_value = map[string]int32{
		"ENROLLED":  0,
		"PROBATION": 1,
	}
)

func (x StudentStatusType) Enum() *StudentStatusType {
	p := new(StudentStatusType)
	*p = x
	return p
}

func (x StudentStatusType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (StudentStatusType) Descriptor() protoreflect.EnumDescriptor {
	return file_student_proto_enumTypes[0].Descriptor()
}

func (StudentStatusType) Type() protoreflect.EnumType {
	return &file_student_proto_enumTypes[0]
}

func (x StudentStatusType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use StudentStatusType.Descriptor instead.
func (StudentStatusType) EnumDescriptor() ([]byte, []int) {
	return file_student_proto_rawDescGZIP(), []int{0, 0}
}

type Student struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id             float64               `protobuf:"fixed64,1,opt,name=id,proto3" json:"id,omitempty"`
	Age            int32                 `protobuf:"varint,2,opt,name=age,proto3" json:"age,omitempty"`
	Gpa            float64               `protobuf:"fixed64,3,opt,name=gpa,proto3" json:"gpa,omitempty"`
	Status         StudentStatusType     `protobuf:"varint,4,opt,name=status,proto3,enum=testdata.school.StudentStatusType" json:"status,omitempty"`
	EnrollmentDate *timestamp.Timestamp  `protobuf:"bytes,7,opt,name=enrollment_date,json=enrollmentDate,proto3" json:"enrollment_date,omitempty"`
	Attrs          map[string]string     `protobuf:"bytes,6,rep,name=attrs,proto3" json:"attrs,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Grades         []float64             `protobuf:"fixed64,5,rep,packed,name=grades,proto3" json:"grades,omitempty"`
	Suspensions    []*Student_Suspension `protobuf:"bytes,8,rep,name=suspensions,proto3" json:"suspensions,omitempty"`
}

func (x *Student) Reset() {
	*x = Student{}
	if protoimpl.UnsafeEnabled {
		mi := &file_student_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Student) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Student) ProtoMessage() {}

func (x *Student) ProtoReflect() protoreflect.Message {
	mi := &file_student_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Student.ProtoReflect.Descriptor instead.
func (*Student) Descriptor() ([]byte, []int) {
	return file_student_proto_rawDescGZIP(), []int{0}
}

func (x *Student) GetId() float64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Student) GetAge() int32 {
	if x != nil {
		return x.Age
	}
	return 0
}

func (x *Student) GetGpa() float64 {
	if x != nil {
		return x.Gpa
	}
	return 0
}

func (x *Student) GetStatus() StudentStatusType {
	if x != nil {
		return x.Status
	}
	return Student_ENROLLED
}

func (x *Student) GetEnrollmentDate() *timestamp.Timestamp {
	if x != nil {
		return x.EnrollmentDate
	}
	return nil
}

func (x *Student) GetAttrs() map[string]string {
	if x != nil {
		return x.Attrs
	}
	return nil
}

func (x *Student) GetGrades() []float64 {
	if x != nil {
		return x.Grades
	}
	return nil
}

func (x *Student) GetSuspensions() []*Student_Suspension {
	if x != nil {
		return x.Suspensions
	}
	return nil
}

type StudentSummary struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Gpa        float64            `protobuf:"fixed64,1,opt,name=gpa,proto3" json:"gpa,omitempty"`
	RiskFactor float64            `protobuf:"fixed64,2,opt,name=risk_factor,json=riskFactor,proto3" json:"risk_factor,omitempty"`
	Tenure     *duration.Duration `protobuf:"bytes,3,opt,name=tenure,proto3" json:"tenure,omitempty"`
}

func (x *StudentSummary) Reset() {
	*x = StudentSummary{}
	if protoimpl.UnsafeEnabled {
		mi := &file_student_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StudentSummary) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StudentSummary) ProtoMessage() {}

func (x *StudentSummary) ProtoReflect() protoreflect.Message {
	mi := &file_student_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StudentSummary.ProtoReflect.Descriptor instead.
func (*StudentSummary) Descriptor() ([]byte, []int) {
	return file_student_proto_rawDescGZIP(), []int{1}
}

func (x *StudentSummary) GetGpa() float64 {
	if x != nil {
		return x.Gpa
	}
	return 0
}

func (x *StudentSummary) GetRiskFactor() float64 {
	if x != nil {
		return x.RiskFactor
	}
	return 0
}

func (x *StudentSummary) GetTenure() *duration.Duration {
	if x != nil {
		return x.Tenure
	}
	return nil
}

type Student_Suspension struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Cause string               `protobuf:"bytes,1,opt,name=cause,proto3" json:"cause,omitempty"`
	Date  *timestamp.Timestamp `protobuf:"bytes,2,opt,name=date,proto3" json:"date,omitempty"`
}

func (x *Student_Suspension) Reset() {
	*x = Student_Suspension{}
	if protoimpl.UnsafeEnabled {
		mi := &file_student_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Student_Suspension) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Student_Suspension) ProtoMessage() {}

func (x *Student_Suspension) ProtoReflect() protoreflect.Message {
	mi := &file_student_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Student_Suspension.ProtoReflect.Descriptor instead.
func (*Student_Suspension) Descriptor() ([]byte, []int) {
	return file_student_proto_rawDescGZIP(), []int{0, 1}
}

func (x *Student_Suspension) GetCause() string {
	if x != nil {
		return x.Cause
	}
	return ""
}

func (x *Student_Suspension) GetDate() *timestamp.Timestamp {
	if x != nil {
		return x.Date
	}
	return nil
}

var File_student_proto protoreflect.FileDescriptor

var file_student_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x73, 0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x0f, 0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x73, 0x63, 0x68, 0x6f, 0x6f, 0x6c,
	0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0x94, 0x04, 0x0a, 0x07, 0x53, 0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x12, 0x0e, 0x0a,
	0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x01, 0x52, 0x02, 0x69, 0x64, 0x12, 0x10, 0x0a,
	0x03, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x03, 0x61, 0x67, 0x65, 0x12,
	0x10, 0x0a, 0x03, 0x67, 0x70, 0x61, 0x18, 0x03, 0x20, 0x01, 0x28, 0x01, 0x52, 0x03, 0x67, 0x70,
	0x61, 0x12, 0x3c, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x24, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x73, 0x63, 0x68,
	0x6f, 0x6f, 0x6c, 0x2e, 0x53, 0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x2e, 0x73, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12,
	0x43, 0x0a, 0x0f, 0x65, 0x6e, 0x72, 0x6f, 0x6c, 0x6c, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x64, 0x61,
	0x74, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x52, 0x0e, 0x65, 0x6e, 0x72, 0x6f, 0x6c, 0x6c, 0x6d, 0x65, 0x6e, 0x74,
	0x44, 0x61, 0x74, 0x65, 0x12, 0x39, 0x0a, 0x05, 0x61, 0x74, 0x74, 0x72, 0x73, 0x18, 0x06, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x73,
	0x63, 0x68, 0x6f, 0x6f, 0x6c, 0x2e, 0x53, 0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x2e, 0x41, 0x74,
	0x74, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x05, 0x61, 0x74, 0x74, 0x72, 0x73, 0x12,
	0x16, 0x0a, 0x06, 0x67, 0x72, 0x61, 0x64, 0x65, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x01, 0x52,
	0x06, 0x67, 0x72, 0x61, 0x64, 0x65, 0x73, 0x12, 0x45, 0x0a, 0x0b, 0x73, 0x75, 0x73, 0x70, 0x65,
	0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x08, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x74,
	0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x73, 0x63, 0x68, 0x6f, 0x6f, 0x6c, 0x2e, 0x53,
	0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x2e, 0x53, 0x75, 0x73, 0x70, 0x65, 0x6e, 0x73, 0x69, 0x6f,
	0x6e, 0x52, 0x0b, 0x73, 0x75, 0x73, 0x70, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x1a, 0x38,
	0x0a, 0x0a, 0x41, 0x74, 0x74, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x52, 0x0a, 0x0a, 0x53, 0x75, 0x73, 0x70,
	0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x14, 0x0a, 0x05, 0x63, 0x61, 0x75, 0x73, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x63, 0x61, 0x75, 0x73, 0x65, 0x12, 0x2e, 0x0a, 0x04,
	0x64, 0x61, 0x74, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x04, 0x64, 0x61, 0x74, 0x65, 0x22, 0x2a, 0x0a, 0x0b,
	0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x12, 0x0c, 0x0a, 0x08, 0x45,
	0x4e, 0x52, 0x4f, 0x4c, 0x4c, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0d, 0x0a, 0x09, 0x50, 0x52, 0x4f,
	0x42, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x10, 0x01, 0x22, 0x76, 0x0a, 0x0e, 0x53, 0x74, 0x75, 0x64,
	0x65, 0x6e, 0x74, 0x53, 0x75, 0x6d, 0x6d, 0x61, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x67, 0x70,
	0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x01, 0x52, 0x03, 0x67, 0x70, 0x61, 0x12, 0x1f, 0x0a, 0x0b,
	0x72, 0x69, 0x73, 0x6b, 0x5f, 0x66, 0x61, 0x63, 0x74, 0x6f, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x01, 0x52, 0x0a, 0x72, 0x69, 0x73, 0x6b, 0x46, 0x61, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x31, 0x0a,
	0x06, 0x74, 0x65, 0x6e, 0x75, 0x72, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x06, 0x74, 0x65, 0x6e, 0x75, 0x72, 0x65,
	0x42, 0x35, 0x5a, 0x33, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65,
	0x7a, 0x61, 0x63, 0x68, 0x72, 0x69, 0x73, 0x65, 0x6e, 0x2f, 0x69, 0x6e, 0x64, 0x69, 0x67, 0x6f,
	0x2f, 0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0x2f, 0x73, 0x63, 0x68, 0x6f, 0x6f, 0x6c,
	0x3b, 0x73, 0x63, 0x68, 0x6f, 0x6f, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_student_proto_rawDescOnce sync.Once
	file_student_proto_rawDescData = file_student_proto_rawDesc
)

func file_student_proto_rawDescGZIP() []byte {
	file_student_proto_rawDescOnce.Do(func() {
		file_student_proto_rawDescData = protoimpl.X.CompressGZIP(file_student_proto_rawDescData)
	})
	return file_student_proto_rawDescData
}

var file_student_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_student_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_student_proto_goTypes = []interface{}{
	(StudentStatusType)(0),      // 0: testdata.school.Student.status_type
	(*Student)(nil),             // 1: testdata.school.Student
	(*StudentSummary)(nil),      // 2: testdata.school.StudentSummary
	nil,                         // 3: testdata.school.Student.AttrsEntry
	(*Student_Suspension)(nil),  // 4: testdata.school.Student.Suspension
	(*timestamp.Timestamp)(nil), // 5: google.protobuf.Timestamp
	(*duration.Duration)(nil),   // 6: google.protobuf.Duration
}
var file_student_proto_depIdxs = []int32{
	0, // 0: testdata.school.Student.status:type_name -> testdata.school.Student.status_type
	5, // 1: testdata.school.Student.enrollment_date:type_name -> google.protobuf.Timestamp
	3, // 2: testdata.school.Student.attrs:type_name -> testdata.school.Student.AttrsEntry
	4, // 3: testdata.school.Student.suspensions:type_name -> testdata.school.Student.Suspension
	6, // 4: testdata.school.StudentSummary.tenure:type_name -> google.protobuf.Duration
	5, // 5: testdata.school.Student.Suspension.date:type_name -> google.protobuf.Timestamp
	6, // [6:6] is the sub-list for method output_type
	6, // [6:6] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_student_proto_init() }
func file_student_proto_init() {
	if File_student_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_student_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Student); i {
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
		file_student_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StudentSummary); i {
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
		file_student_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Student_Suspension); i {
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
			RawDescriptor: file_student_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_student_proto_goTypes,
		DependencyIndexes: file_student_proto_depIdxs,
		EnumInfos:         file_student_proto_enumTypes,
		MessageInfos:      file_student_proto_msgTypes,
	}.Build()
	File_student_proto = out.File
	file_student_proto_rawDesc = nil
	file_student_proto_goTypes = nil
	file_student_proto_depIdxs = nil
}
