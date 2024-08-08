package service

import (
	"context" // เรียกใช้งาน package context เพื่อจัดการกับบริบทของการเรียก RPC

	"grpc-project/example.com/pcbook/pb" // import protobuf generated code
	"google.golang.org/grpc/codes" // เรียกใช้งาน package codes เพื่อใช้รหัสสถานะของ gRPC
	"google.golang.org/grpc/status" // เรียกใช้งาน package status เพื่อสร้างและจัดการกับสถานะของ gRPC
)

// AuthServer is the server for authentication
// AuthServer เป็นโครงสร้างเซิร์ฟเวอร์สำหรับการรับรองความถูกต้อง
type AuthServer struct {
	pb.UnimplementedAuthServiceServer // Embedding struct เพื่อให้ง่ายต่อการขยายในอนาคต
	userStore  UserStore // อินเตอร์เฟซสำหรับการจัดการข้อมูลผู้ใช้
	jwtManager *JWTManager // ตัวจัดการ JWT สำหรับสร้างและตรวจสอบ token
}

// NewAuthServer returns a new auth server
// NewAuthServer คืนค่า instance ใหม่ของ AuthServer
func NewAuthServer(userStore UserStore, jwtManager *JWTManager) pb.AuthServiceServer {
	return &AuthServer{userStore: userStore, jwtManager: jwtManager} // สร้างและคืนค่า AuthServer ใหม่
}

// Login is a unary RPC to login user
// Login เป็น unary RPC สำหรับการเข้าสู่ระบบของผู้ใช้
func (server *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// ค้นหาผู้ใช้โดยใช้ชื่อผู้ใช้จาก request
	user, err := server.userStore.Find(req.GetUsername())
	if err != nil {
		// ถ้าค้นหาไม่สำเร็จ ให้คืนค่า error พร้อมกับรหัสสถานะ Internal
		return nil, status.Errorf(codes.Internal, "cannot find user: %v", err)
	}

	if user == nil || !user.IsCorrectPassword(req.GetPassword()) {
		// ถ้าผู้ใช้ไม่พบหรือรหัสผ่านไม่ถูกต้อง ให้คืนค่า error พร้อมกับรหัสสถานะ NotFound
		return nil, status.Errorf(codes.NotFound, "incorrect username/password")
	}

	// สร้าง token ใหม่สำหรับผู้ใช้
	token, err := server.jwtManager.Generate(user)
	if err != nil {
		// ถ้าการสร้าง token ล้มเหลว ให้คืนค่า error พร้อมกับรหัสสถานะ Internal
		return nil, status.Errorf(codes.Internal, "cannot generate access token")
	}

	// สร้าง response ที่มี access token
	res := &pb.LoginResponse{AccessToken: token}
	return res, nil // คืนค่า response และ nil แทน error
}
