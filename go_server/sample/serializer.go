package sample

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// WriteProtobufToBinaryFile writes a proto.Message to a binary file
func WriteProtobufToBinaryFile(message proto.Message, filename string) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("proto.Marshal: %v", err)
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("ioutil.WriteFile: %v", err)
	}
	return nil
}

// ReadProtobufFromBinaryFile reads a proto.Message from a binary file
func ReadProtobufFromBinaryFile(message proto.Message, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("ioutil.ReadFile: %v", err)
	}
	if err := proto.Unmarshal(data, message); err != nil {
		return fmt.Errorf("proto.Unmarshal: %v", err)
	}
	return nil
}

// WriteProtobufToJSONFile writes a proto.Message to a JSON file
func WriteProtobufToJSONFile(message proto.Message, filename string) error {
	data, err := protojson.MarshalOptions{
		Indent:            "  ",
		UseProtoNames:     true,
		UseEnumNumbers:    false,
		EmitDefaultValues: true,
		EmitUnpopulated:   true,
	}.Marshal(message)
	if err != nil {
		return fmt.Errorf("protojson.Marshal: %v", err)
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("ioutil.WriteFile: %v", err)
	}
	return nil
}

// ReadProtobufFromJSONFile reads a proto.Message from a JSON file
func ReadProtobufFromJSONFile(message proto.Message, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("ioutil.ReadFile: %v", err)
	}
	if err := protojson.Unmarshal(data, message); err != nil {
		return fmt.Errorf("protojson.Unmarshal: %v", err)
	}
	return nil
}
