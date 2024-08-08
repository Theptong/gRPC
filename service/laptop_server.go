package service

import (
	"bytes"
	"context"
	"errors"
	"grpc-project/example.com/pcbook/pb"
	"io"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
)

const maxImageSize = 1 << 20 // ขนาดสูงสุดของภาพที่สามารถอัปโหลดได้ (1MB)

// LaptopServer เป็นการ implement เมธอดของ pb.LaptopServiceServer
type LaptopServer struct {
	pb.UnimplementedLaptopServiceServer             // ฝัง struct ที่ไม่ได้ implement เพื่อไม่ต้อง implement ทุกเมธอด
	laptopStore                         LaptopStore // ตัวแปรสำหรับเก็บข้อมูลแล็ปท็อป
	imageStore                          ImageStore  // ตัวแปรสำหรับเก็บข้อมูลภาพ
	ratingStore                         RatingStore // ตัวแปรสำหรับเก็บข้อมูลการจัดอันดับ
}

// NewLaptopServer สร้าง instance ใหม่ของ LaptopServer
func NewLaptopServer(store LaptopStore, image ImageStore, rating RatingStore) *LaptopServer {
	return &LaptopServer{
		laptopStore: store,
		imageStore:  image,
		ratingStore: rating,
	}
}

// CreateLaptop เป็นฟังก์ชันที่จัดการการสร้างแล็ปท็อปใหม่
func (server *LaptopServer) CreateLaptop(ctx context.Context, req *pb.CreateLaptopRequest) (*pb.CreateLaptopResponse, error) {
	laptop := req.GetLaptop() // ดึงข้อมูลแล็ปท็อปจากคำขอ
	log.Printf("ได้รับคำขอการสร้างแล็ปท็อปพร้อม ID: %s", laptop.Id)

	// ตรวจสอบว่ามีการให้ ID ของแล็ปท็อปหรือไม่
	if len(laptop.Id) > 0 {
		_, err := uuid.Parse(laptop.Id) // ตรวจสอบว่า ID ที่ให้มานั้นเป็น UUID ที่ถูกต้องหรือไม่
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "ID ของแล็ปท็อปไม่ใช่ UUID ที่ถูกต้อง: %v", err)
		}
	} else {
		id, err := uuid.NewRandom() // สร้าง UUID ใหม่หากไม่มีการให้ ID
		if err != nil {
			return nil, status.Errorf(codes.Internal, "ไม่สามารถสร้าง ID ใหม่สำหรับแล็ปท็อป: %v", err)
		}
		laptop.Id = id.String() // ตั้งค่า ID ใหม่ให้กับแล็ปท็อป
	}

	// ตรวจสอบ context ว่าถูกยกเลิกหรือหมดเวลาหรือไม่
	if ctx.Err() == context.Canceled {
		return nil, status.Error(codes.Canceled, "request is Canceled")
	}
	if ctx.Err() == context.DeadlineExceeded {
		log.Print("deadline is exceeded")
		return nil, status.Error(codes.DeadlineExceeded, "deadline is exceeded")
	}

	// บันทึกแล็ปท็อปลงใน store
	err := server.laptopStore.Save(laptop)
	code := codes.Internal
	if errors.Is(err, ErrAlreadyExists) {
		code = codes.AlreadyExists // ตั้งค่า error code เป็น AlreadyExists หากแล็ปท็อปมีอยู่แล้ว
	}
	if err != nil {
		return nil, status.Errorf(code, "ไม่สามารถบันทึกแล็ปท็อปลงใน store: %v", err)
	}

	log.Printf("บันทึกแล็ปท็อปด้วย ID: %s", laptop.Id)

	// คืนค่าการตอบสนองที่ประสบความสำเร็จพร้อม ID ของแล็ปท็อป
	res := &pb.CreateLaptopResponse{
		Id: laptop.Id,
	}
	return res, nil
}

// SearchLaptop เป็นฟังก์ชันที่จัดการการค้นหาแล็ปท็อปตามเงื่อนไขที่ระบุ
func (server *LaptopServer) SearchLaptop(
	req *pb.SearchLaptopRequest,
	stream pb.LaptopService_SearchLaptopServer,
) error {
	filter := req.GetFilter() // ดึงข้อมูลเงื่อนไขในการค้นหาแล็ปท็อปจากคำขอ
	log.Printf("Received a search-laptop request with filter: %v", filter)

	// เรียกใช้ฟังก์ชัน Search ของ laptopStore เพื่อค้นหาแล็ปท็อปตามเงื่อนไขที่ระบุ
	err := server.laptopStore.Search(
		stream.Context(),
		filter,
		func(laptop *pb.Laptop) error {
			// ตรวจสอบ context ก่อนการส่งข้อมูล
			if stream.Context().Err() != nil {
				return stream.Context().Err()
			}

			res := &pb.SearchLaptopResponse{Laptop: laptop}
			err := stream.Send(res) // ส่งข้อมูลแล็ปท็อปที่พบไปยังไคลเอนต์
			if err != nil {
				return err
			}
			log.Printf("Sent laptop with id: %s", laptop.GetId())
			return nil
		},
	)
	if err != nil {
		return status.Errorf(codes.Internal, "unexpected error: %v", err)
	}
	return nil
}

// UploadImage เป็นฟังก์ชันที่จัดการการอัปโหลดภาพของแล็ปท็อป
func (server *LaptopServer) UploadImage(stream pb.LaptopService_UploadImageServer) error {
	req, err := stream.Recv() // รับคำขอแรกจากไคลเอนต์
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot receive image info"))
	}

	laptopID := req.GetInfo().GetLaptopId()   // ดึงข้อมูล ID ของแล็ปท็อปจากคำขอ
	imageType := req.GetInfo().GetImageType() // ดึงประเภทของภาพจากคำขอ
	log.Printf("receive an upload-image request for laptop %s with image type %s", laptopID, imageType)

	// ตรวจสอบว่าแล็ปท็อปมีอยู่ใน store หรือไม่
	laptop, err := server.laptopStore.Find(laptopID)
	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannot find laptop: %v", err))
	}
	if laptop == nil {
		return logError(status.Errorf(codes.InvalidArgument, "laptop id %s doesn't exist", laptopID))
	}

	imageData := bytes.Buffer{} // ตัวแปรสำหรับเก็บข้อมูลภาพ
	imageSize := 0              // ตัวแปรสำหรับเก็บขนาดของภาพ

	// รับข้อมูลภาพเป็นชิ้นๆ จากไคลเอนต์
	for {
		err := contextError(stream.Context()) // ตรวจสอบ context ว่าถูกยกเลิกหรือหมดเวลาหรือไม่
		if err != nil {
			return err
		}

		log.Print("waiting to receive more data")

		req, err := stream.Recv() // รับข้อมูลภาพเป็นชิ้นๆ
		if err == io.EOF {
			log.Print("no more data")
			break
		}
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot receive chunk data: %v", err))
		}

		chunk := req.GetChunkData() // ดึงข้อมูลชิ้นภาพจากคำขอ
		size := len(chunk)

		log.Printf("received a chunk with size: %d", size)

		imageSize += size
		if imageSize > maxImageSize {
			return logError(status.Errorf(codes.InvalidArgument, "image is too large: %d > %d", imageSize, maxImageSize))
		}

		_, err = imageData.Write(chunk) // เขียนข้อมูลชิ้นภาพลงในตัวแปร imageData
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot write chunk data: %v", err))
		}
	}

	// บันทึกภาพลงใน store
	imageID, err := server.imageStore.Save(laptopID, imageType, imageData)
	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannot save image to the store: %v", err))
	}

	res := &pb.UploadImageResponse{
		Id:   imageID,
		Size: uint32(imageSize),
	}

	err = stream.SendAndClose(res) // ส่งการตอบสนองและปิดสตรีม
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot send response: %v", err))
	}

	log.Printf("saved image with id: %s, size: %d", imageID, imageSize)
	return nil
}

// RateLaptop เป็นฟังก์ชันที่จัดการการให้คะแนนแล็ปท็อป
func (server *LaptopServer) RateLaptop(stream pb.LaptopService_RateLaptopServer) error {
	for {
		err := contextError(stream.Context()) // ตรวจสอบ context ว่าถูกยกเลิกหรือหมดเวลาหรือไม่
		if err != nil {
			return err
		}

		req, err := stream.Recv() // รับคำขอจากไคลเอนต์
		if err == io.EOF {
			log.Print("no more data")
			break
		}
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot receive stream request: %v", err))
		}

		laptopID := req.GetLaptopId() // ดึงข้อมูล ID ของแล็ปท็อปจากคำขอ
		score := req.GetScore()       // ดึงคะแนนจากคำขอ

		log.Printf("received a rate-laptop request: id = %s, score = %.2f", laptopID, score)

		// ตรวจสอบว่าแล็ปท็อปมีอยู่ใน store หรือไม่
		found, err := server.laptopStore.Find(laptopID)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot find laptop: %v", err))
		}
		if found == nil {
			return logError(status.Errorf(codes.NotFound, "laptopID %s is not found", laptopID))
		}

		// เพิ่มคะแนนลงใน store
		rating, err := server.ratingStore.Add(laptopID, score)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot add rating to the store: %v", err))
		}

		res := &pb.RateLaptopResponse{
			LaptopId:     laptopID,
			RatedCount:   rating.Count,
			AverageScore: rating.Sum / float64(rating.Count), // คำนวณคะแนนเฉลี่ย
		}

		err = stream.Send(res) // ส่งการตอบสนองไปยังไคลเอนต์
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot send stream response: %v", err))
		}
	}

	return nil
}

// contextError ตรวจสอบ context ว่าถูกยกเลิกหรือหมดเวลาหรือไม่
func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return logError(status.Error(codes.Canceled, "request is canceled"))
	case context.DeadlineExceeded:
		return logError(status.Error(codes.DeadlineExceeded, "deadline is exceeded"))
	default:
		return nil
	}
}

// logError บันทึกข้อผิดพลาดลงใน log และคืนค่าข้อผิดพลาด
func logError(err error) error {
	if err != nil {
		log.Print(err)
	}
	return err
}
