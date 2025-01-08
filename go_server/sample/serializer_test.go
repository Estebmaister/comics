package sample

import (
	"comics/pb"
	"os"
	"testing"

	"google.golang.org/protobuf/proto"
)

var binaryTempFile = "./comic.temp.bin"
var jsonTempFile = "./comic.temp.json"

func TestSerializerFile(t *testing.T) {
	t.Parallel()

	comicCreated := NewComic()
	err := WriteProtobufToBinaryFile(comicCreated, binaryTempFile)
	if err != nil {
		t.Errorf("WriteProtobufToBinaryFile() error = %v", err)
		return
	}

	comicFromBinary := &pb.Comic{}
	err = ReadProtobufFromBinaryFile(comicFromBinary, binaryTempFile)
	if err != nil {
		t.Errorf("ReadProtobufFromBinaryFile() error = %v", err)
		return
	}

	if !proto.Equal(comicCreated, comicFromBinary) {
		t.Errorf("comicCreated != comicFromBinary")
		return
	}

	err = WriteProtobufToJSONFile(comicCreated, jsonTempFile)
	if err != nil {
		t.Errorf("WriteProtobufToJSONFile() error = %v", err)
		return
	}

	comicFromJSON := &pb.Comic{}
	err = ReadProtobufFromJSONFile(comicFromJSON, jsonTempFile)
	if err != nil {
		t.Errorf("ReadProtobufFromJSONFile() error = %v", err)
		return
	}

	if !proto.Equal(comicCreated, comicFromJSON) {
		t.Errorf("comicCreated != comicFromJSON")
		return
	}

	if !proto.Equal(comicFromBinary, comicFromJSON) {
		t.Errorf("comicFromBinary != comicFromJSON")
		return
	}

	err = os.Remove(binaryTempFile)
	if err != nil {
		t.Errorf("os.Remove() error = %v", err)
		return
	}
	err = os.Remove(jsonTempFile)
	if err != nil {
		t.Errorf("os.Remove() error = %v", err)
		return
	}
}
