package sampler

import (
	"os"
	"testing"

	pb "comics/pkg/pb"

	"google.golang.org/protobuf/proto"
)

const (
	binaryTempFile = "./comic.temp.bin"
	jsonTempFile   = "./comic.temp.json"
	badTempFile    = "./temp/badfile"
	wrongFile      = "./serializer.go"
)

func TestSerializerFileWithComic(t *testing.T) {
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

func TestFailBadFileSerializer(t *testing.T) {
	t.Parallel()

	err := WriteProtobufToBinaryFile(nil, badTempFile)
	if err == nil {
		t.Errorf("WriteProtobufToBinaryFile(nil, badfile) error shouldn't be nil")
		return
	}

	err = ReadProtobufFromBinaryFile(nil, badTempFile)
	if err == nil {
		t.Errorf("ReadProtobufFromBinaryFile(nil, badfile) error shouldn't be nil")
		return
	}

	err = WriteProtobufToJSONFile(nil, badTempFile)
	if err == nil {
		t.Errorf("WriteProtobufToJSONFile(nil, badfile) error shouldn't be nil")
		return
	}

	err = ReadProtobufFromJSONFile(nil, badTempFile)
	if err == nil {
		t.Errorf("ReadProtobufFromJSONFile(nil, badfile) error shouldn't be nil")
		return
	}
}

func TestFailBadProtoSerializer(t *testing.T) {
	t.Parallel()

	emptyProto := &pb.Comic{}
	err := ReadProtobufFromBinaryFile(emptyProto, wrongFile)
	if err == nil {
		t.Errorf("ReadProtobufFromBinaryFile(nil, badfile) error shouldn't be nil")
		return
	}

	err = ReadProtobufFromJSONFile(emptyProto, wrongFile)
	if err == nil {
		t.Errorf("ReadProtobufFromJSONFile(nil, badfile) error shouldn't be nil")
		return
	}
}
