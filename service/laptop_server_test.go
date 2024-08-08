package service_test

import (
	"context"
	"grpc-project/example.com/pcbook/pb"
	"grpc-project/sample"
	"grpc-project/service"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestServerCreateLaptop เป็นฟังก์ชันการทดสอบหลักที่ใช้ทดสอบฟังก์ชัน CreateLaptop ของ LaptopServer
func TestServerCreateLaptop(t *testing.T) {
	t.Parallel() // ให้การทดสอบทำงานแบบขนานเพื่อเพิ่มความเร็วในการทดสอบ

	// สร้างตัวอย่าง Laptop ที่ไม่มี ID
	laptopNoID := sample.NewLaptop()
	laptopNoID.Id = ""

	// สร้างตัวอย่าง Laptop ที่มี ID เป็น UUID ที่ไม่ถูกต้อง
	laptopInvalidID := sample.NewLaptop()
	laptopInvalidID.Id = "invalid-uuid"

	// สร้างตัวอย่าง Laptop ที่มี ID ซ้ำกัน และเก็บไว้ใน storeDuplicateID
	laptopDuplicateID := sample.NewLaptop()
	storeDuplicateID := service.NewInMemoryLaptopStore()
	err := storeDuplicateID.Save(laptopDuplicateID) // บันทึก Laptop ที่มี ID ไม่ถูกต้อง
	require.Nil(t, err)                           // ตรวจสอบว่าไม่มีข้อผิดพลาดในการบันทึก

	// กำหนดกรณีทดสอบต่าง ๆ
	testCases := []struct {
		name   string
		laptop *pb.Laptop
		store  service.LaptopStore
		code   codes.Code // รหัสข้อผิดพลาดที่คาดหวัง
	}{
		{
			name:   "success_with_id", // กรณีที่มี ID ที่ถูกต้อง
			laptop: sample.NewLaptop(),
			store:  service.NewInMemoryLaptopStore(),
			code:   codes.OK,
		},
		{
			name:   "success_no_id", // กรณีที่ไม่มี ID และควรสร้าง ID ใหม่
			laptop: laptopNoID,
			store:  service.NewInMemoryLaptopStore(),
			code:   codes.OK,
		},
		{
			name:   "failure_invalid_id", // กรณีที่มี ID ที่ไม่ถูกต้อง
			laptop: laptopInvalidID,
			store:  service.NewInMemoryLaptopStore(),
			code:   codes.InvalidArgument,
		},
		{
			name:   "failure_duplicate_id", // กรณีที่มี ID ซ้ำกัน
			laptop: laptopDuplicateID,
			store:  storeDuplicateID,
			code:   codes.AlreadyExists,
		},
	}

	// ทำการทดสอบตามกรณีที่กำหนดไว้
	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // ให้แต่ละกรณีทดสอบทำงานแบบขนาน

			// สร้าง request สำหรับการทดสอบ
			req := &pb.CreateLaptopRequest{
				Laptop: tc.laptop,
			}

			// สร้าง instance ของ LaptopServer โดยใช้ store ที่กำหนดในกรณีทดสอบ
			server := service.NewLaptopServer(tc.store,nil,nil)

			// เรียกใช้ CreateLaptop และตรวจสอบผลลัพธ์
			res, err := server.CreateLaptop(context.Background(), req)
			if tc.code == codes.OK {
				// สำหรับกรณีที่คาดหวังผลลัพธ์เป็น OK
				require.NoError(t, err)     // ตรวจสอบว่าไม่มีข้อผิดพลาด
				require.NotNil(t, res)      // ตรวจสอบว่าผลลัพธ์ไม่เป็น nil
				require.NotEmpty(t, res.Id) // ตรวจสอบว่า ID ของ laptop ไม่ว่างเปล่า
				if len(tc.laptop.Id) > 0 {
					require.Equal(t, tc.laptop.Id, res.Id) // ตรวจสอบว่า ID ตรงกับที่ส่งไป
				}
			} else {
				// สำหรับกรณีที่คาดหวังข้อผิดพลาด
				require.Error(t, err)                // ตรวจสอบว่ามีข้อผิดพลาดเกิดขึ้น
				st, ok := status.FromError(err)      // แปลงข้อผิดพลาดเป็น status
				require.True(t, ok)                  // ตรวจสอบว่าการแปลงสำเร็จ
				require.Equal(t, tc.code, st.Code()) // ตรวจสอบรหัสสถานะของข้อผิดพลาด
			}
		})
	}
}
