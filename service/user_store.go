package service

import "sync" // เรียกใช้งาน package sync เพื่อใช้ mutex สำหรับการจัดการ concurrent access

// UserStore is an interface to store users
// UserStore เป็น interface สำหรับการเก็บข้อมูลผู้ใช้
type UserStore interface {
	// Save saves a user to the store
	// Save เก็บข้อมูลผู้ใช้ไปยัง store
	Save(user *User) error
	// Find finds a user by username
	// Find ค้นหาผู้ใช้โดยใช้ชื่อผู้ใช้
	Find(username string) (*User, error)
}

// InMemoryUserStore stores users in memory
// InMemoryUserStore เป็นโครงสร้างที่เก็บข้อมูลผู้ใช้ในหน่วยความจำ
type InMemoryUserStore struct {
	mutex sync.RWMutex // ใช้ mutex เพื่อป้องกันการเข้าถึงข้อมูลพร้อมกัน
	users map[string]*User // แผนที่ที่เก็บข้อมูลผู้ใช้ โดยใช้ชื่อผู้ใช้เป็นกุญแจ
}

// NewInMemoryUserStore returns a new in-memory user store
// NewInMemoryUserStore คืนค่า instance ใหม่ของ InMemoryUserStore
func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		users: make(map[string]*User), // สร้างแผนที่ใหม่สำหรับเก็บข้อมูลผู้ใช้
	}
}

// Save saves a user to the store
// Save เก็บข้อมูลผู้ใช้ไปยัง store
func (store *InMemoryUserStore) Save(user *User) error {
	store.mutex.Lock() // ล็อค mutex เพื่อป้องกันการเข้าถึงพร้อมกัน
	defer store.mutex.Unlock() // ปลดล็อค mutex เมื่อฟังก์ชันทำงานเสร็จ

	if store.users[user.Username] != nil {
		// ถ้าผู้ใช้ที่มีชื่อผู้ใช้นี้มีอยู่แล้ว ให้คืนค่า error
		return ErrAlreadyExists
	}

	store.users[user.Username] = user.Clone() // เก็บข้อมูลผู้ใช้ที่ถูกคัดลอกไปยังแผนที่
	return nil // คืนค่า nil แทน error
}

// Find finds a user by username
// Find ค้นหาผู้ใช้โดยใช้ชื่อผู้ใช้
func (store *InMemoryUserStore) Find(username string) (*User, error) {
	store.mutex.RLock() // ล็อค mutex เพื่อป้องกันการเข้าถึงพร้อมกันแบบอ่านอย่างเดียว
	defer store.mutex.RUnlock() // ปลดล็อค mutex เมื่อฟังก์ชันทำงานเสร็จ

	user := store.users[username] // ค้นหาผู้ใช้จากแผนที่
	if user == nil {
		// ถ้าผู้ใช้ไม่พบ ให้คืนค่า nil และ error
		return nil, nil
	}

	return user.Clone(), nil // คืนค่าผู้ใช้ที่ถูกคัดลอก และ nil แทน error
}
