package service

import (
	"fmt" // เรียกใช้งาน package fmt เพื่อใช้ฟังก์ชันการจัดการ string เช่น Errorf

	"golang.org/x/crypto/bcrypt" // เรียกใช้งาน package bcrypt เพื่อเข้ารหัสและตรวจสอบรหัสผ่าน
)

// User contains user's information
// โครงสร้าง User เก็บข้อมูลผู้ใช้
type User struct {
	Username       string // ชื่อผู้ใช้
	HashedPassword string // รหัสผ่านที่ถูกเข้ารหัส
	Role           string // บทบาทของผู้ใช้
}

// NewUser returns a new user
// ฟังก์ชัน NewUser สร้างและคืนค่า User ใหม่
func NewUser(username string, password string, role string) (*User, error) {
	// เข้ารหัสรหัสผ่านโดยใช้ bcrypt และระดับความยากที่ตั้งค่าไว้
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// ถ้าเกิดข้อผิดพลาดในการเข้ารหัส ให้คืนค่า nil และ error ที่ถูกห่อหุ้มด้วย fmt.Errorf
		return nil, fmt.Errorf("cannot hash password: %w", err)
	}

	// สร้าง instance ของ User ด้วยข้อมูลที่ได้รับมา
	user := &User{
		Username:       username,
		HashedPassword: string(hashedPassword), // แปลงรหัสผ่านที่เข้ารหัสเป็น string
		Role:           role,
	}

	// คืนค่า User ใหม่ และ nil แทน error
	return user, nil
}

// IsCorrectPassword checks if the provided password is correct or not
// ฟังก์ชัน IsCorrectPassword ตรวจสอบว่ารหัสผ่านที่ให้มาตรงกับรหัสผ่านที่เข้ารหัสหรือไม่
func (user *User) IsCorrectPassword(password string) bool {
	// ใช้ bcrypt เพื่อเปรียบเทียบรหัสผ่านที่ให้มากับรหัสผ่านที่เข้ารหัส
	err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password))
	return err == nil // ถ้าไม่มีข้อผิดพลาด แปลว่ารหัสผ่านตรงกัน และคืนค่า true
}

// Clone returns a clone of this user
// ฟังก์ชัน Clone คืนค่า clone ของ User นี้
func (user *User) Clone() *User {
	// สร้างและคืนค่า instance ใหม่ของ User โดยคัดลอกข้อมูลจาก User ปัจจุบัน
	return &User{
		Username:       user.Username,
		HashedPassword: user.HashedPassword,
		Role:           user.Role,
	}
}
