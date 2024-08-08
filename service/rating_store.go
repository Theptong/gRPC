package service

import "sync"

// RatingStore is an interface to store laptop ratings
type RatingStore interface {
	// Add adds a new laptop score to the store and returns its rating
	Add(laptopID string, score float64) (*Rating, error)
}

// Rating contains the rating information of a laptop
type Rating struct {
	Count uint32    // Count เก็บจำนวนครั้งที่มีการให้คะแนน
	Sum   float64   // Sum เก็บผลรวมของคะแนนทั้งหมด
}

// InMemoryRatingStore stores laptop ratings in memory
type InMemoryRatingStore struct {
	mutex  sync.RWMutex        // mutex เพื่อใช้ในการจัดการความปลอดภัยของการเข้าถึงข้อมูลพร้อมกัน
	rating map[string]*Rating  // rating เป็น map ที่เก็บข้อมูลคะแนนโดยใช้ laptopID เป็น key
}

// NewInMemoryRatingStore returns a new InMemoryRatingStore
func NewInMemoryRatingStore() *InMemoryRatingStore {
	return &InMemoryRatingStore{
		rating: make(map[string]*Rating),  // สร้าง InMemoryRatingStore ใหม่พร้อมกำหนด map เปล่าๆ
	}
}

// Add adds a new laptop score to the store and returns its rating
func (store *InMemoryRatingStore) Add(laptopID string, score float64) (*Rating, error) {
	store.mutex.Lock()         // ล็อก mutex เพื่อป้องกันการเข้าถึงข้อมูลพร้อมกัน
	defer store.mutex.Unlock() // ปลดล็อก mutex หลังจากทำงานเสร็จ

	rating := store.rating[laptopID]  // ดึงข้อมูล rating ของ laptopID นี้ออกมา
	if rating == nil {                // ถ้า rating ยังไม่มีอยู่
		rating = &Rating{             // สร้าง rating ใหม่
			Count: 1,                  // ตั้งค่า Count เป็น 1
			Sum:   score,              // ตั้งค่า Sum เป็นค่า score ที่รับเข้ามา
		}
	} else {                          // ถ้า rating มีอยู่แล้ว
		rating.Count++                // เพิ่มค่า Count ขึ้น 1
		rating.Sum += score           // เพิ่มค่า Sum ด้วย score ที่รับเข้ามา
	}

	store.rating[laptopID] = rating   // เก็บ rating ที่อัปเดตแล้วกลับเข้าไปใน map

	return rating, nil                // ส่งค่า rating ที่อัปเดตแล้วกลับไป
}
