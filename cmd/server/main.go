package main

import (
	"context" // เรียกใช้งาน package context เพื่อจัดการกับ context ของการเรียกใช้งาน RPC
	"flag"    // เรียกใช้งาน package flag เพื่อจัดการกับ command-line flag
	"fmt"     // เรียกใช้งาน package fmt เพื่อใช้ฟังก์ชัน Println และ Printf
	"log"     // เรียกใช้งาน package log เพื่อ logging
	"net"     // เรียกใช้งาน package net เพื่อใช้ net.Listen สำหรับเปิด port
	"time"    // เรียกใช้งาน package time เพื่อการจัดการกับเวลา

	"grpc-project/example.com/pcbook/pb" // import protobuf generated code
	"grpc-project/service"               // import local service package

	"google.golang.org/grpc" // เรียกใช้งาน package grpc สำหรับการทำ gRPC
	"google.golang.org/grpc/reflection"
	// เรียกใช้งาน package reflection สำหรับการทำ gRPC reflection
)

// unaryInterceptor เป็นฟังก์ชันสำหรับจัดการกับ unary RPC calls (เรียกใช้งานแบบเรียกเดียวและตอบเดียว)
func unaryInterceptor(
	ctx context.Context, // context ของการเรียกใช้งาน RPC
	req interface{}, // request ที่ถูกส่งเข้ามา
	info *grpc.UnaryServerInfo, // ข้อมูลเกี่ยวกับ unary RPC
	handler grpc.UnaryHandler, // handler ของ unary RPC ที่จะถูกเรียกใช้ภายหลัง interceptor
) (interface{}, error) { // ฟังก์ชันนี้จะคืนค่า response และ error
	log.Println("--> unary interceptor: ", info.FullMethod) // log ข้อมูลเกี่ยวกับ method ที่ถูกเรียกใช้งาน
	return handler(ctx, req)                                // เรียก handler เพื่อดำเนินการ RPC และคืนค่าผลลัพธ์
}

// streamInterceptor เป็นฟังก์ชันสำหรับจัดการกับ stream RPC calls (เรียกใช้งานแบบ stream)
func streamInterceptor(
	srv interface{}, // เซิร์ฟเวอร์ที่กำลังถูกเรียกใช้งาน
	stream grpc.ServerStream, // server stream ที่ถูกส่งเข้ามา
	info *grpc.StreamServerInfo, // ข้อมูลเกี่ยวกับ stream RPC
	handler grpc.StreamHandler) error { // handler ของ stream RPC ที่จะถูกเรียกใช้ภายหลัง interceptor
	log.Println("--> stream interceptor: ", info.FullMethod) // log ข้อมูลเกี่ยวกับ method ที่ถูกเรียกใช้งาน
	return handler(srv, stream)                              // เรียก handler เพื่อดำเนินการ RPC และคืนค่า error ถ้ามี
}

// seedUsers เป็นฟังก์ชันเพื่อสร้างผู้ใช้เริ่มต้นใน user store
func seedUsers(userStore service.UserStore) error {
	err := createUser(userStore, "admin1", "secret", "admin")
	if err != nil {
		return err
	}
	return createUser(userStore, "user1", "secret", "user")
}

// createUser เป็นฟังก์ชันเพื่อสร้างผู้ใช้ใหม่และบันทึกลงใน user store
func createUser(userStore service.UserStore, username, password, role string) error {
	user, err := service.NewUser(username, password, role)
	if err != nil {
		return err
	}
	return userStore.Save(user)
}

const (
	secretKey     = "secret"         // ค่าคีย์ลับสำหรับ JWT
	tokenDuration = 15 * time.Minute // ระยะเวลาหมดอายุของ JWT
)

func accessibleRoles() map[string][]string {
	const laptopServicePath = "/techschool.pcbook.LaptopService/"

	return map[string][]string{
		laptopServicePath + "CreateLaptop": {"admin"},
		laptopServicePath + "UploadImage":  {"admin"},
		laptopServicePath + "RateLaptop":   {"admin", "user"},
	}
}

func main() {
	fmt.Println("Hello World from server") // พิมพ์ Hello World from server ออกทาง console

	// ตั้งค่าพอร์ตจาก flag
	port := flag.Int("port", 0, "the server port") // กำหนด flag -port โดยใช้ Int และคำอธิบาย
	flag.Parse()                                   // แปลง command-line flag เป็นค่าตัวแปร

	// ตรวจสอบว่าพอร์ตได้รับการตั้งค่า
	if *port == 0 {
		log.Fatal("port must be set") // ถ้าพอร์ตยังไม่ได้ตั้งค่า ให้ log.Fatal และปิดโปรแกรม
	}

	log.Printf("start server on port %d", *port) // บันทึก log ว่าเริ่มเซิร์ฟเวอร์ที่พอร์ตที่ตั้งค่าไว้

	userStore := service.NewInMemoryUserStore() // สร้าง in-memory user store

	err := seedUsers(userStore) // เริ่มต้นผู้ใช้ใน user store
	if err != nil {
		log.Fatal("cannot seed users: ", err) // ถ้ามี error ในการสร้างผู้ใช้ ให้ log.Fatal และปิดโปรแกรม
	}

	jwtManager := service.NewJWTManager(secretKey, tokenDuration) // สร้าง JWT manager
	authServer := service.NewAuthServer(userStore, jwtManager)    // สร้าง authentication server

	// สร้างเซิร์ฟเวอร์ gRPC
	laptopStore := service.NewInMemoryLaptopStore()                               // สร้าง in-memory store สำหรับเก็บข้อมูล laptop
	imageStore := service.NewDiskImageStore("img")                                // สร้าง disk image store สำหรับเก็บรูปภาพ
	ratingStore := service.NewInMemoryRatingStore()                               // สร้าง in-memory store สำหรับเก็บ rating
	laptopServer := service.NewLaptopServer(laptopStore, imageStore, ratingStore) // สร้าง instance ของ gRPC server โดยใช้ stores ที่สร้างขึ้นมา
	interceptor := service.NewAuthInterceptor(jwtManager, accessibleRoles())
	// serverOptions := []grpc.ServerOption{
	// 	grpc.UnaryInterceptor(interceptor.Unary()),
	// 	grpc.StreamInterceptor(interceptor.Stream()),
	// }
	
	// สร้างเซิร์ฟเวอร์ gRPC
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.Unary()),   // ใช้ unary interceptor
		grpc.StreamInterceptor(interceptor.Stream()), // ใช้ stream interceptor
	)

	pb.RegisterAuthServiceServer(grpcServer, authServer)     // ลงทะเบียน authentication service
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer) // ลงทะเบียน laptop service
	

	// ลงทะเบียน Reflection Service
	reflection.Register(grpcServer)

	// สร้าง listener และเริ่มฟัง
	address := fmt.Sprintf("0.0.0.0:%d", *port) // สร้าง address string จากพอร์ตที่ตั้งค่าไว้
	listener, err := net.Listen("tcp", address) // เปิด listener ด้วย address ที่สร้างขึ้น
	if err != nil {
		log.Fatalf("cannot start listener: %v", err) // ถ้ามี error ในการเปิด listener ให้ log.Fatal และปิดโปรแกรม
	}

	// เริ่มเซิร์ฟเวอร์
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("cannot start server: %v", err) // ถ้ามี error ในการเริ่มเซิร์ฟเวอร์ ให้ log.Fatal และปิดโปรแกรม
	}
}
