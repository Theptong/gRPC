package service_test

import (
	"bufio"
	"context"
	"fmt"
	"grpc-project/example.com/pcbook/pb"
	"grpc-project/sample"
	"grpc-project/serializer"
	"grpc-project/service"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"google.golang.org/grpc"
)

// ทดสอบการสร้างแล็ปท็อปโดยใช้ client
func TestClientCreateLaptop(t *testing.T) {
	t.Parallel() // ทำการทดสอบใน parallel
	laptopStore := service.NewInMemoryLaptopStore() // สร้าง store ในหน่วยความจำสำหรับเก็บแล็ปท็อป
	serverAddress := startTestLaptopServer(t, laptopStore, nil, nil) // เริ่มเซิร์ฟเวอร์ทดสอบ
	laptopClient := newTestLaptopClient(t, serverAddress) // สร้าง client สำหรับทดสอบ

	laptop := sample.NewLaptop() // สร้างแล็ปท็อปใหม่
	expectedID := laptop.Id // เก็บ ID ของแล็ปท็อปที่คาดหวัง
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}
	res, err := laptopClient.CreateLaptop(context.Background(), req) // ส่งคำขอสร้างแล็ปท็อปไปยังเซิร์ฟเวอร์
	require.NoError(t, err) // ตรวจสอบว่าการเรียกใช้งานสำเร็จ
	require.NotNil(t, res) // ตรวจสอบว่าผลลัพธ์ไม่เป็น nil
	require.Equal(t, expectedID, res.Id) // ตรวจสอบว่า ID ที่สร้างตรงกับที่คาดหวัง

	other, err := laptopStore.Find(res.Id) // ค้นหาแล็ปท็อปใน store ตาม ID
	require.NoError(t, err) // ตรวจสอบว่าการค้นหาสำเร็จ
	require.NotNil(t, other) // ตรวจสอบว่าผลลัพธ์ไม่เป็น nil

	requireSameLaptop(t, laptop, other) // ตรวจสอบว่าแล็ปท็อปที่เก็บใน store ตรงกับที่สร้าง
}

// ทดสอบการค้นหาแล็ปท็อปตามเงื่อนไขที่ระบุ
func TestClientSearchLaptop(t *testing.T) {
	t.Parallel() // ทำการทดสอบใน parallel

	filter := &pb.Filter{
		MaxPriceUsd: 2000, // กำหนดเงื่อนไขการค้นหา
		MinCpuCores: 4,
		MinCpuGhz:   2.2,
		MinRam:      &pb.Memory{Value: 8, Unit: pb.Memory_GIGABYTE},
	}
	laptopStore := service.NewInMemoryLaptopStore() // สร้าง store ในหน่วยความจำสำหรับเก็บแล็ปท็อป
	expectedIDs := make(map[string]bool) // สร้างแผนที่สำหรับเก็บ ID ที่คาดหวัง

	// สร้างแล็ปท็อปทดสอบและบันทึกใน store
	for i := 0; i < 6; i++ {
		laptop := sample.NewLaptop()
		switch i {
		case 0:
			laptop.PriceUsd = 2500
		case 1:
			laptop.Cpu.NumberCores = 2
		case 2:
			laptop.Cpu.MinGhz = 2.0
		case 3:
			laptop.Ram = &pb.Memory{Value: 4096, Unit: pb.Memory_GIGABYTE}
		case 4:
			laptop.PriceUsd = 1999
			laptop.Cpu.NumberCores = 4
			laptop.Cpu.MinGhz = 2.5
			laptop.Cpu.MaxGhz = 4.5
			laptop.Ram = &pb.Memory{Value: 16, Unit: pb.Memory_GIGABYTE}
			expectedIDs[laptop.Id] = true // เก็บ ID ของแล็ปท็อปที่ตรงตามเงื่อนไข
		case 5:
			laptop.PriceUsd = 2000
			laptop.Cpu.NumberCores = 6
			laptop.Cpu.MinGhz = 2.8
			laptop.Cpu.MaxGhz = 5.0
			laptop.Ram = &pb.Memory{Value: 64, Unit: pb.Memory_GIGABYTE}
			expectedIDs[laptop.Id] = true // เก็บ ID ของแล็ปท็อปที่ตรงตามเงื่อนไข
		}
		err := laptopStore.Save(laptop) // บันทึกแล็ปท็อปใน store
		require.NoError(t, err) // ตรวจสอบว่าการบันทึกสำเร็จ
	}
	serverAddress := startTestLaptopServer(t, laptopStore, nil, nil) // เริ่มเซิร์ฟเวอร์ทดสอบ
	laptopClient := newTestLaptopClient(t, serverAddress) // สร้าง client สำหรับทดสอบ

	req := &pb.SearchLaptopRequest{Filter: filter} // สร้างคำขอค้นหาแล็ปท็อป
	stream, err := laptopClient.SearchLaptop(context.Background(), req) // ส่งคำขอค้นหาและรับ stream ของผลลัพธ์
	require.NoError(t, err) // ตรวจสอบว่าการเรียกใช้งานสำเร็จ

	found := 0
	for {
		res, err := stream.Recv() // รับข้อมูลแล็ปท็อปจาก stream
		if err == io.EOF {
			break // สิ้นสุดการรับข้อมูล
		}
		require.NoError(t, err) // ตรวจสอบว่าไม่มีข้อผิดพลาด
		require.Contains(t, expectedIDs, res.GetLaptop().GetId()) // ตรวจสอบว่า ID ของแล็ปท็อปที่พบตรงกับที่คาดหวัง
		found += 1
	}
	require.Equal(t, len(expectedIDs), found) // ตรวจสอบว่าจำนวนแล็ปท็อปที่พบตรงกับที่คาดหวัง
}

// ทดสอบการอัปโหลดภาพของแล็ปท็อป
func TestClientUploadImage(t *testing.T) {
	t.Parallel() // ทำการทดสอบใน parallel

	testImageFolder := "../tmp" // โฟลเดอร์ที่เก็บภาพ

	laptopStore := service.NewInMemoryLaptopStore() // สร้าง store ในหน่วยความจำสำหรับเก็บแล็ปท็อป
	imageStore := service.NewDiskImageStore(testImageFolder) // สร้าง store สำหรับเก็บภาพในดิสก์

	laptop := sample.NewLaptop() // สร้างแล็ปท็อปใหม่
	err := laptopStore.Save(laptop) // บันทึกแล็ปท็อปใน store
	require.NoError(t, err) // ตรวจสอบว่าการบันทึกสำเร็จ

	serverAddress := startTestLaptopServer(t, laptopStore, imageStore, nil) // เริ่มเซิร์ฟเวอร์ทดสอบ
	laptopClient := newTestLaptopClient(t, serverAddress) // สร้าง client สำหรับทดสอบ

	imagePath := fmt.Sprintf("%s/laptop.jpg", testImageFolder) // เส้นทางของภาพที่ต้องการอัปโหลด
	file, err := os.Open(imagePath) // เปิดไฟล์ภาพ
	require.NoError(t, err) // ตรวจสอบว่าเปิดไฟล์สำเร็จ
	defer file.Close() // ปิดไฟล์เมื่อเสร็จสิ้น

	stream, err := laptopClient.UploadImage(context.Background()) // สร้าง stream สำหรับอัปโหลดภาพ
	require.NoError(t, err) // ตรวจสอบว่าการสร้าง stream สำเร็จ

	imageType := filepath.Ext(imagePath) // นามสกุลของภาพ
	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.ImageInfo{
				LaptopId:  laptop.GetId(),
				ImageType: imageType,
			},
		},
	}

	err = stream.Send(req) // ส่งข้อมูลภาพเริ่มต้น (ข้อมูลเมตา) ไปยังเซิร์ฟเวอร์
	require.NoError(t, err) // ตรวจสอบว่าการส่งข้อมูลสำเร็จ

	reader := bufio.NewReader(file) // สร้าง reader สำหรับอ่านไฟล์ภาพ
	buffer := make([]byte, 1024) // สร้าง buffer สำหรับอ่านข้อมูลภาพเป็นชิ้นๆ
	size := 0

	for {
		n, err := reader.Read(buffer) // อ่านข้อมูลภาพจากไฟล์
		if err == io.EOF {
			break // สิ้นสุดการอ่านข้อมูล
		}

		require.NoError(t, err) // ตรวจสอบว่าไม่มีข้อผิดพลาด
		size += n // เพิ่มขนาดข้อมูลที่อ่าน

		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n], // ข้อมูลชิ้นส่วนของภาพ
			},
		}

		err = stream.Send(req) // ส่งข้อมูลภาพเป็นชิ้นๆ ไปยังเซิร์ฟเวอร์
		require.NoError(t, err) // ตรวจสอบว่าการส่งข้อมูลสำเร็จ
	}

	res, err := stream.CloseAndRecv() // ปิด stream และรับการตอบสนองจากเซิร์ฟเวอร์
	require.NoError(t, err) // ตรวจสอบว่าการรับการตอบสนองสำเร็จ
	require.NotZero(t, res.GetId()) // ตรวจสอบว่ามี ID ของภาพที่ได้รับ
	require.EqualValues(t, size, res.GetSize()) // ตรวจสอบขนาดของภาพที่อัปโหลด

	savedImagePath := fmt.Sprintf("%s/%s%s", testImageFolder, res.GetId(), imageType) // เส้นทางที่บันทึกภาพ
	require.FileExists(t, savedImagePath) // ตรวจสอบว่าภาพถูกบันทึกสำเร็จ
	require.NoError(t, os.Remove(savedImagePath)) // ลบภาพที่บันทึกแล้วออก
}

// ทดสอบการให้คะแนนแล็ปท็อป
func TestClientRateLaptop(t *testing.T) {
	t.Parallel() // ทำการทดสอบใน parallel

	laptopStore := service.NewInMemoryLaptopStore() // สร้าง store ในหน่วยความจำสำหรับเก็บแล็ปท็อป
	ratingStore := service.NewInMemoryRatingStore() // สร้าง store ในหน่วยความจำสำหรับเก็บคะแนน

	laptop := sample.NewLaptop() // สร้างแล็ปท็อปใหม่
	err := laptopStore.Save(laptop) // บันทึกแล็ปท็อปใน store
	require.NoError(t, err) // ตรวจสอบว่าการบันทึกสำเร็จ

	serverAddress := startTestLaptopServer(t, laptopStore, nil, ratingStore) // เริ่มเซิร์ฟเวอร์ทดสอบ
	laptopClient := newTestLaptopClient(t, serverAddress) // สร้าง client สำหรับทดสอบ

	stream, err := laptopClient.RateLaptop(context.Background()) // สร้าง stream สำหรับให้คะแนนแล็ปท็อป
	require.NoError(t, err) // ตรวจสอบว่าการสร้าง stream สำเร็จ

	scores := []float64{8, 7.5, 10} // คะแนนที่ต้องการให้
	averages := []float64{8, 7.75, 8.5} // คะแนนเฉลี่ยที่คาดหวัง
	n := len(scores)
	for i := 0; i < n; i++ {
		req := &pb.RateLaptopRequest{
			LaptopId: laptop.GetId(),
			Score:    scores[i],
		}
		err := stream.Send(req) // ส่งคะแนนให้เซิร์ฟเวอร์
		require.NoError(t, err) // ตรวจสอบว่าการส่งคะแนนสำเร็จ
	}

	err = stream.CloseSend() // ปิด stream การส่งคะแนน
	require.NoError(t, err) // ตรวจสอบว่าการปิด stream สำเร็จ
	
	for idx := 0; ; idx++ {
		res, err := stream.Recv() // รับการตอบสนองจากเซิร์ฟเวอร์
		if err == io.EOF {
			require.Equal(t, n, idx) // ตรวจสอบว่าจำนวนคะแนนที่ได้รับตรงกับที่คาดหวัง
			return
		}
		require.NoError(t, err) // ตรวจสอบว่าไม่มีข้อผิดพลาด
		require.Equal(t, laptop.GetId(), res.GetLaptopId()) // ตรวจสอบ ID ของแล็ปท็อป
		require.Equal(t, uint32(idx+1), res.GetRatedCount()) // ตรวจสอบจำนวนคะแนนที่ได้รับ
		require.Equal(t, averages[idx], res.GetAverageScore()) // ตรวจสอบคะแนนเฉลี่ยที่คาดหวัง
	}
}

// ฟังก์ชันสำหรับเริ่มเซิร์ฟเวอร์ทดสอบ
func startTestLaptopServer(t *testing.T, laptopStore service.LaptopStore, imageStore service.ImageStore, ratingStore service.RatingStore) string {
	laptopServer := service.NewLaptopServer(laptopStore, imageStore, ratingStore) // สร้างเซิร์ฟเวอร์แล็ปท็อป

	grpcServer := grpc.NewServer() // สร้าง gRPC server
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer) // ลงทะเบียนบริการแล็ปท็อปกับ gRPC server

	listener, err := net.Listen("tcp", ":0") // ฟังบนพอร์ตที่ว่าง
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	go func() {
		if err := grpcServer.Serve(listener); err != nil { // เริ่มเซิร์ฟเวอร์
			t.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	return listener.Addr().String() // คืนค่า address ของเซิร์ฟเวอร์
}

// ฟังก์ชันสำหรับสร้าง client ทดสอบ
func newTestLaptopClient(t *testing.T, serverAddress string) pb.LaptopServiceClient {
	conn, err := grpc.Dial(serverAddress, grpc.WithInsecure()) // สร้างการเชื่อมต่อกับเซิร์ฟเวอร์
	require.NoError(t, err)
	return pb.NewLaptopServiceClient(conn) // คืนค่า client ของบริการแล็ปท็อป
}

// ฟังก์ชันสำหรับตรวจสอบว่าแล็ปท็อปสองตัวเหมือนกัน
func requireSameLaptop(t *testing.T, laptop1 *pb.Laptop, laptop2 *pb.Laptop) {
	json1, err := serializer.ProtobufToJson(laptop1) // แปลง protobuf เป็น JSON
	require.NoError(t, err)

	json2, err := serializer.ProtobufToJson(laptop2) // แปลง protobuf เป็น JSON
	require.NoError(t, err)

	require.Equal(t, json1, json2) // ตรวจสอบว่า JSON ของแล็ปท็อปทั้งสองตัวเหมือนกัน
}
