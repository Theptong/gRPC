package service

import (
	"context"
	"errors"
	"fmt"
	"grpc-project/example.com/pcbook/pb"
	"log"
	"sync"
	"time"

	"github.com/jinzhu/copier"
)

// ErrAlreadyExists เป็นข้อผิดพลาดที่ระบุว่าข้อมูลที่พยายามจะบันทึกมีอยู่แล้วใน store
var ErrAlreadyExists = errors.New("record already exists")

// LaptopStore เป็น interface ที่กำหนดว่า struct ใดๆ ที่ต้องการทำหน้าที่เกี่ยวกับการจัดเก็บข้อมูลแล็ปท็อป
// จะต้องมีฟังก์ชัน Save ที่รับพารามิเตอร์เป็น pointer ของ pb.Laptop และส่งคืน error หากเกิดปัญหา
type LaptopStore interface {
	Save(laptop *pb.Laptop) error
	Find(id string) (*pb.Laptop, error)
	Search(ctx context.Context,filter *pb.Filter, found func(laptop *pb.Laptop) error) error
}

// InMemoryLaptopStore เป็น struct ที่ใช้สำหรับเก็บข้อมูลแล็ปท็อปในหน่วยความจำ
// มีฟิลด์ mutex เพื่อจัดการกับการ Lock/Unlock ข้อมูล และฟิลด์ data ที่เก็บข้อมูลของแล็ปท็อปในรูปแบบของ map
type InMemoryLaptopStore struct {
	mutex sync.RWMutex          // ใช้สำหรับการ Lock/Unlock ข้อมูลเพื่อความปลอดภัยของการเข้าถึงพร้อมกัน
	data  map[string]*pb.Laptop // เก็บข้อมูลแล็ปท็อปโดยใช้ ID เป็นคีย์
}

// NewInMemoryLaptopStore สร้าง instance ใหม่ของ InMemoryLaptopStore และทำการตั้งค่าเริ่มต้น
func NewInMemoryLaptopStore() *InMemoryLaptopStore {
	return &InMemoryLaptopStore{
		data: make(map[string]*pb.Laptop), // สร้าง map ว่างสำหรับเก็บข้อมูลแล็ปท็อป
	}
}

// Save ฟังก์ชันสำหรับบันทึกข้อมูลแล็ปท็อปลงใน store
func (store *InMemoryLaptopStore) Save(laptop *pb.Laptop) error {
	store.mutex.Lock() // ทำการ Lock ข้อมูลเพื่อป้องกันการเข้าถึงพร้อมกัน

	defer store.mutex.Unlock() // ทำการ Unlock ข้อมูลเมื่อฟังก์ชันเสร็จสิ้น

	// ตรวจสอบว่ามีแล็ปท็อปที่มี ID เดียวกันใน store หรือไม่
	if store.data[laptop.Id] != nil {
		return ErrAlreadyExists // ส่งคืนข้อผิดพลาดหากมีแล็ปท็อปที่มี ID เดียวกันอยู่แล้ว
	}

	// ทำการ deep copy ข้อมูลของแล็ปท็อปเพื่อป้องกันปัญหาการแก้ไขข้อมูลที่ส่งเข้ามา
	other, err := deepCopy(laptop)
	if err != nil {
		return err // ส่งคืนข้อผิดพลาดหากไม่สามารถทำการ copy ได้
	}

	// บันทึกข้อมูลแล็ปท็อปใน map ด้วย ID เป็นคีย์
	store.data[other.Id] = other
	return nil // คืนค่าผลลัพธ์เป็น nil เพื่อแสดงว่าการบันทึกสำเร็จ
}

func (store *InMemoryLaptopStore) Find(id string) (*pb.Laptop, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	laptop := store.data[id]
	if laptop == nil {
		return nil, nil
	}
	return deepCopy(laptop)
}

func (store *InMemoryLaptopStore) Search(
    ctx context.Context,
    filter *pb.Filter,
    found func(laptop *pb.Laptop) error,
) error {
    store.mutex.RLock()
    defer store.mutex.RUnlock()

    for _, laptop := range store.data {
        time.Sleep(time.Second) // เลียนแบบการทำงานที่ใช้เวลานาน
        log.Print("checking laptop id: ", laptop.GetId())

        if ctx.Err() == context.Canceled || ctx.Err() == context.DeadlineExceeded {
            log.Print("context is cancelled")
            return ctx.Err() // คืนค่า context error
        }

        if isQualified(filter, laptop) {
            other, err := deepCopy(laptop)
            if err != nil {
                return fmt.Errorf("failed to copy laptop: %w", err)
            }

            err = found(other)
            if err != nil {
                return fmt.Errorf("error calling found function: %w", err)
            }
        }
    }

    return nil
}




func isQualified(filter *pb.Filter, laptop *pb.Laptop) bool {
	if laptop.GetPriceUsd() > filter.GetMaxPriceUsd() {
		return false
	}
	if laptop.GetCpu().GetNumberCores() < filter.GetMinCpuCores() {
		return false
	}
	if laptop.GetCpu().GetMinGhz() < filter.GetMinCpuGhz() {
		return false
	}
	if toBit(laptop.GetRam()) < toBit(filter.GetMinRam()) {
		return false
	}
	return true
}

func toBit(memory *pb.Memory) uint64 {
	value := memory.GetValue()

	switch memory.GetUnit() {
	case pb.Memory_BIT:
		return value
	case pb.Memory_BYTE:
		return value * 8 // 8= 2^3
	case pb.Memory_KILOBYTE:
		return value << 13 // 1024* 8 = 2^10 *2^3 = 2^13
	case pb.Memory_MEGABYTE:
		return value << 23
	case pb.Memory_GIGABYTE:
		return value << 33
	case pb.Memory_TERABYTE:
		return value << 43
	default:
		return 0
	}
}

func deepCopy(laptop *pb.Laptop) (*pb.Laptop, error) {
	// deep copy
	other := &pb.Laptop{}
	err := copier.Copy(other, laptop)
	if err != nil {
		return nil, fmt.Errorf("cannot copy laptop data: %w", err)
	}
	return other, nil
}
