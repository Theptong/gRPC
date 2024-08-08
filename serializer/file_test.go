package serializer_test

import (
	"grpc-project/sample"
	"grpc-project/serializer"
	"testing"
	"google.golang.org/protobuf/proto"
	"github.com/stretchr/testify/require"
	"grpc-project/example.com/pcbook/pb"
)

func TestFileSerializer(t *testing.T) {
	t.Parallel() // รันการทดสอบในโหมด parallel

	binaryFile := "../tmp/laptop.bin" // ไฟล์ไบนารีที่ใช้ในการทดสอบ
	jsonFile := "../tmp/laptop.json"   // ไฟล์ JSON ที่ใช้ในการทดสอบ

	// สร้างตัวอย่าง laptop1
	laptop1 := sample.NewLaptop()

	// เขียน laptop1 ลงไฟล์ไบนารี
	err := serializer.WriteProtobufToBinaryFile(laptop1, binaryFile)
	require.NoError(t, err) // ตรวจสอบว่าไม่มีข้อผิดพลาด

	// สร้างตัวแปร laptop2 เพื่ออ่านข้อมูลจากไฟล์ไบนารี
	laptop2 := &pb.Laptop{}
	err = serializer.ReadProtobuffFromBinaryFile(binaryFile, laptop2)
	require.NoError(t, err) // ตรวจสอบว่าไม่มีข้อผิดพลาด

	// ตรวจสอบว่า laptop1 และ laptop2 เท่ากัน
	require.True(t, proto.Equal(laptop1, laptop2))

	// เขียน laptop1 ลงไฟล์ JSON
	err = serializer.WriteProtobufToJsonFile(laptop1, jsonFile)
	require.NoError(t, err) // ตรวจสอบว่าไม่มีข้อผิดพลาด
}
