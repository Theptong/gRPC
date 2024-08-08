package serializer

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"
)
// WriteProtobufToJsonFile write protocal buffer message to Json file
func WriteProtobufToJsonFile(message proto.Message, filename string) error{
	data, err := ProtobufToJson(message)
	if err != nil {
		return fmt.Errorf("cannot marshal proto message to JSON: %w", err)
	}
	err = os.WriteFile(filename, []byte(data), 0644)
	if err != nil{
		return fmt.Errorf("cannot write JSON data to file: %w", err)
	}
	return nil
}

// WriteProtobufToBinaryFile write protocal buffer message to binary file
func WriteProtobufToBinaryFile(message proto.Message, filename string) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("cannot marshal proto message to binary:%w", err)
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("cannot write binary data to binary:%w", err)
	}
	return nil
}

// ReadProtobuffFromBinaryFile read protocal buffer message to binary file
func ReadProtobuffFromBinaryFile(filename string, message proto.Message) error{
	data,err := os.ReadFile(filename)
	if err != nil{
		return fmt.Errorf("cannot read proto message to binary:%w", err)
	}
	err = proto.Unmarshal(data,message)
	if err != nil{
		return fmt.Errorf("cannot unmarshal binary to proto message:%w", err)
	}
	return nil
}