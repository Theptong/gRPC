package serializer

import (
    "github.com/golang/protobuf/jsonpb"   // นำเข้าpackage jsonpb สำหรับการแปลง Protocol Buffers เป็น JSON
    "google.golang.org/protobuf/proto"    // นำเข้าpackage proto สำหรับการทำงานกับ Protocol Buffers
    "google.golang.org/protobuf/types/known/anypb" // นำเข้าพackage anypb สำหรับการแปลงเป็นประเภทที่รองรับ jsonpb
)

// ProtobufToJson converts protocol buffer message to JSON string
func ProtobufToJson(message proto.Message) (string, error) {
    // แปลง proto.Message เป็นประเภท anypb.Any ซึ่งเป็นประเภทที่รองรับการใช้กับ jsonpb
    anyMessage, err := anypb.New(message)
    if err != nil {
        // ถ้ามีข้อผิดพลาดในการแปลงเป็น anypb.Any คืนค่า "" และ error
        return "", err
    }

    // สร้าง Marshaler ของ jsonpb สำหรับการแปลงเป็น JSON
    marshaler := jsonpb.Marshaler{
        EnumsAsInts:  true,   // แปลง enums เป็นตัวเลข (integer) แทนการแสดงชื่อของ enum
        EmitDefaults: true,  // แสดงค่าที่เป็น default ของฟิลด์ที่ไม่ได้ถูกตั้งค่า
        Indent:      " ",    // จัดรูปแบบ JSON ให้อ่านง่ายด้วยการเว้นวรรค
        OrigName:    true,   // ใช้ชื่อฟิลด์ตามที่กำหนดใน proto file แทนการแปลงเป็น camelCase
    }
    
    // แปลง anypb.Any เป็น JSON string โดยใช้ marshaler
    jsonData, err := marshaler.MarshalToString(anyMessage)
    if err != nil {
        // ถ้ามีข้อผิดพลาดในการแปลงเป็น JSON คืนค่า "" และ error
        return "", err
    }
    
    // คืนค่า JSON string ที่แปลงได้
    return jsonData, nil
}
