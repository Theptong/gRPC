package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"grpc-project/client"
	"grpc-project/example.com/pcbook/pb"
	"grpc-project/sample"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testCreateLaptop ทดสอบการสร้างแล็ปท็อปใหม่
func testCreateLaptop(laptopClient *client.LaptopClient) {
	laptopClient.CreateLaptop(sample.NewLaptop())
}

// testSearchLaptop ทดสอบการค้นหาแล็ปท็อปตามเงื่อนไข
func testSearchLaptop(laptopClient *client.LaptopClient) {
	for i := 0; i < 10; i++ {
		laptopClient.CreateLaptop(sample.NewLaptop()) // สร้างแล็ปท็อปใหม่ 10 เครื่องเพื่อทดสอบการค้นหา
	}

	filter := &pb.Filter{
		MaxPriceUsd: 3000,
		MinCpuCores: 4,
		MinCpuGhz:   2.5,
		MinRam:      &pb.Memory{Value: 8, Unit: pb.Memory_GIGABYTE},
	}
	laptopClient.SearchLaptop(filter) // ค้นหาแล็ปท็อปตามเงื่อนไขที่ระบุ
}

// testUploadImage ทดสอบการอัปโหลดภาพของแล็ปท็อป
func testUploadImage(laptopClient *client.LaptopClient) {
	laptop := sample.NewLaptop()
	laptopClient.CreateLaptop(laptop)
	laptopClient.UploadImage(laptop.GetId(), "tmp/laptop.jpg")
}

func testRateLaptop(laptopClient *client.LaptopClient) {
	n := 3
	laptopIDs := make([]string, n)

	for i := 0; i < n; i++ {
		laptop := sample.NewLaptop()
		laptopIDs[i] = laptop.GetId()
		laptopClient.CreateLaptop(laptop)
	}

	scores := make([]float64, n)
	for {
		fmt.Print("rate laptop (y/n)? ")
		var answer string
		fmt.Scan(&answer)

		if strings.ToLower(answer) != "y" {
			break
		}

		for i := 0; i < n; i++ {
			scores[i] = sample.RandomLaptopScore()
		}

		err := laptopClient.RateLaptop(laptopIDs, scores)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// createLaptop สร้างแล็ปท็อปใหม่ในเซิร์ฟเวอร์
func createLaptop(laptopClient pb.LaptopServiceClient, laptop *pb.Laptop) string {
	// laptop := sample.NewLaptop()
	// laptop.Id = ""
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // สร้าง context พร้อม timeout 10 วินาที
	defer cancel()

	res, err := laptopClient.CreateLaptop(ctx, req) // ส่งคำขอสร้างแล็ปท็อปไปยังเซิร์ฟเวอร์
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			log.Print("laptop already exists")
		} else {
			log.Fatal("connot create laptop: ", err)
		}
		return ""
	}
	log.Printf("created laptop with id: %s", res.Id)
	return res.Id
}

// searchLaptop ค้นหาแล็ปท็อปตามเงื่อนไขที่ระบุ
func searchLaptop(laptopClient pb.LaptopServiceClient, filter *pb.Filter) {
	log.Print("search filter: ", filter)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // สร้าง context พร้อม timeout 5 วินาที
	defer cancel()

	req := &pb.SearchLaptopRequest{Filter: filter}
	stream, err := laptopClient.SearchLaptop(ctx, req) // ส่งคำขอค้นหาแล็ปท็อปไปยังเซิร์ฟเวอร์และรับ stream ผลลัพธ์
	if err != nil {
		log.Fatal("cannot search laptop: ", err)
	}

	for {
		res, err := stream.Recv() // รับข้อมูลแล็ปท็อปจาก stream
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatal("cannot receive response: ", err)
		}

		laptop := res.GetLaptop()
		log.Print("- found: ", laptop.GetId())
		log.Print(" + brand: ", laptop.GetBrand())
		log.Print(" + name: ", laptop.GetName())
		log.Print(" + cpu cores: ", laptop.GetCpu().GetNumberCores())
		log.Print(" + cpu min ghz: ", laptop.GetCpu().MinGhz)
		log.Print(" + ram: ", laptop.GetRam().GetValue(), laptop.GetRam().GetUnit())
	}
}

// uploadImage อัปโหลดภาพของแล็ปท็อปไปยังเซิร์ฟเวอร์
func uploadImage(laptopClient pb.LaptopServiceClient, laptopID string, imagePath string) {
	file, err := os.Open(imagePath) // เปิดไฟล์ภาพ
	if err != nil {
		log.Fatal("cannot open image file: ", err)
	}
	defer file.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // สร้าง context พร้อม timeout 5 วินาที
	defer cancel()

	stream, err := laptopClient.UploadImage(ctx) // สร้าง stream สำหรับอัปโหลดภาพ
	if err != nil {
		log.Fatal("cannot open image file: ", err)
	}
	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.ImageInfo{
				LaptopId:  laptopID,
				ImageType: filepath.Ext(imagePath),
			},
		},
	}
	err = stream.Send(req) // ส่งข้อมูลภาพเริ่มต้น (ข้อมูลเมตา)
	if err != nil {
		log.Fatal("cannot send image info: ", err, stream.RecvMsg(nil))
	}

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024) // สร้างบัฟเฟอร์สำหรับอ่านข้อมูลภาพเป็นชิ้นๆ

	for {
		n, err := reader.Read(buffer) // อ่านข้อมูลภาพจากไฟล์
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("cannot read chunk to buffer: ", err)
		}
		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}
		err = stream.Send(req) // ส่งข้อมูลภาพเป็นชิ้นๆ ไปยังเซิร์ฟเวอร์
		if err != nil {
			log.Fatal("cannot send chunk to server: ", err, stream.RecvMsg(nil))
		}
	}

	res, err := stream.CloseAndRecv() // ปิด stream และรับการตอบสนองจากเซิร์ฟเวอร์
	if err != nil {
		log.Fatal("cannot receive response: ", err)
	}

	log.Printf("image uploaded with id: %s, size: %d", res.GetId(), res.GetSize())
}

func rateLaptop(laptopClient pb.LaptopServiceClient, laptopIDs []string, scores []float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // สร้าง context พร้อม timeout 5 วินาที
	defer cancel()
	stream, err := laptopClient.RateLaptop(ctx)
	if err != nil {
		return fmt.Errorf("cannot rate laptop: %v", err)
	}
	waitResponse := make(chan error)
	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				log.Print("no more responses")
				waitResponse <- nil
				return
			}
			if err != nil {
				waitResponse <- fmt.Errorf("cannot receive stream response: %v", err)
				return
			}
			log.Print("received response: ", res)
		}
	}()
	for i, laptopID := range laptopIDs {
		req := &pb.RateLaptopRequest{
			LaptopId: laptopID,
			Score:    scores[i],
		}
		err := stream.Send(req)
		if err != nil {
			fmt.Errorf("cannot sead stream request: %v - %v", err, stream.RecvMsg(nil))
		}
		log.Print("sent  request: ", req)
	}
	err = stream.CloseSend()
	if err != nil {
		return fmt.Errorf("cannot close send: %v", err)
	}
	err = <-waitResponse
	return err
}


const (
	username        = "admin1"
	password        = "secret"
	refreshDuration = 30 * time.Second
)

func authMethods() map[string]bool {
	const laptopServicePath = "/techschool.pcbook.LaptopService/"

	return map[string]bool{
		laptopServicePath + "CreateLaptop": true,
		laptopServicePath + "UploadImage":  true,
		laptopServicePath + "RateLaptop":   true,
	}
}

func main() {
	fmt.Println("Hello World client")
	serverAddress := flag.String("address", "", "the server address") // อ่านที่อยู่เซิร์ฟเวอร์จาก flag
	flag.Parse()
	log.Printf("dial server %s", *serverAddress)

	cc1, err := grpc.Dial(*serverAddress, grpc.WithInsecure()) // เชื่อมต่อไปยังเซิร์ฟเวอร์
	if err != nil {
		log.Fatalf("cannot connect to server: %v", err)
	}
	authClient := client.NewAuthClient(cc1,username,password)
	interceptor, err := client.NewAuthInterceptor(authClient, authMethods(), refreshDuration)
	
	cc2, err := grpc.Dial(
		*serverAddress,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(interceptor.Unary()),
		grpc.WithStreamInterceptor(interceptor.Stream()),
	)
	if err != nil {
		log.Fatal("cannot dial server: ", err)
	}

	laptopClient := client.NewLaptopClient(cc2)
	// laptopClient := pb.NewLaptopServiceClient(conn) // สร้าง client สำหรับเรียกใช้บริการ LaptopService
	testRateLaptop(laptopClient)                   // เรียกใช้ฟังก์ชันทดสอบการอัปโหลดภาพ
}
