package parsing_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/jonathanlloyd/skewserver/parsing"
)

// We want to simulate the fact that reads can return any number of bytes
// and can cut frames into arbritrary segments
var READ_SIZES_BYTES = [...]int{1, 8, 32, 12, 5, 2}

// Should return proper struct from single frame

// No body, no headers
func TestSingleFrameNoBodyNoHeaders(t *testing.T) {
	testData := "CONNECT\n\n\x00"

	conn := mockTCPStream{streamData: testData}
	parser := parsing.NewStompParserFromReader(&conn)
	frame, err := parser.NextFrame()

	if err != nil {
		t.Errorf("No error should be raised")
	}

	if frame.Command != parsing.CONNECT {
		t.Errorf("Frame type should have type CONNECT")
	}

	expectedHeaders := map[string]string{}
	if !reflect.DeepEqual(expectedHeaders, frame.Headers) {
		t.Errorf("Frame should have no headers")
	}

	expectedBody := []byte{}
	if !bytes.Equal(expectedBody, frame.Body) {
		t.Errorf("Frame should have no body")
	}
}

// No body, with headers
func TestSingleFrameNoBodyWithHeaders(t *testing.T) {
	testData := "CONNECT\naccept-version:1.2\n\n\x00"

	conn := mockTCPStream{streamData: testData}
	parser := parsing.NewStompParserFromReader(&conn)
	frame, err := parser.NextFrame()

	if err != nil {
		t.Errorf("No error should be raised")
	}

	if frame.Command != parsing.CONNECT {
		t.Errorf("Frame type should have type CONNECT")
	}

	expectedHeaders := map[string]string{
		"accept-version": "1.2",
	}
	if !reflect.DeepEqual(expectedHeaders, frame.Headers) {
		t.Errorf("Frame should have correct headers")
	}

	expectedBody := []byte{}
	if !bytes.Equal(expectedBody, frame.Body) {
		t.Errorf("Frame should have no body")
	}
}

// With body, with headers
func TestSingleFrameWithBodyWithHeaders(t *testing.T) {
	testData := "MESSAGE\nx-custom-header:some value\n\nmessage body\x00"

	conn := mockTCPStream{streamData: testData}
	parser := parsing.NewStompParserFromReader(&conn)
	frame, err := parser.NextFrame()

	if err != nil {
		t.Errorf("No error should be raised")
	}

	if frame.Command != parsing.MESSAGE {
		t.Errorf("Frame type should have type CONNECT")
	}

	expectedHeaders := map[string]string{
		"x-custom-header": "some value",
	}
	if !reflect.DeepEqual(expectedHeaders, frame.Headers) {
		t.Errorf("Frame should have correct headers")
	}

	expectedBody := []byte("message body")
	if !bytes.Equal(expectedBody, frame.Body) {
		t.Errorf("Frame should have correct body")
	}
}

// Trailing EOLs
func TestTrailingEOLs(t *testing.T) {
	testData := "MESSAGE\nx-custom-header:some value\n\nmessage body\x00\n\n\nMESSAGE\nx-custom-header:some value\n\nmessage body\x00"

	conn := mockTCPStream{streamData: testData}
	parser := parsing.NewStompParserFromReader(&conn)
	frame, err := parser.NextFrame()
	frame, err = parser.NextFrame()

	if err != nil {
		t.Errorf("No error should be raised")
	}

	if frame.Command != parsing.MESSAGE {
		t.Errorf("Frame type should have type CONNECT")
	}

	expectedHeaders := map[string]string{
		"x-custom-header": "some value",
	}
	if !reflect.DeepEqual(expectedHeaders, frame.Headers) {
		t.Errorf("Frame should have correct headers")
	}

	expectedBody := []byte("message body")
	if !bytes.Equal(expectedBody, frame.Body) {
		t.Errorf("Frame should have correct body")
	}
}

// Multiple frames
func TestMultipleFrames(t *testing.T) {
	testData := "CONNECT\naccept-version:1.2\n\n\x00CONNECTED\r\nversion:1.2\n\n\x00"

	conn := mockTCPStream{streamData: testData}
	parser := parsing.NewStompParserFromReader(&conn)
	frame, err := parser.NextFrame()

	if err != nil {
		t.Errorf("No error should be raised for frame 1")
	}

	if frame.Command != parsing.CONNECT {
		t.Errorf("Frame 1 should have type CONNECT")
	}

	expectedHeaders := map[string]string{
		"accept-version": "1.2",
	}
	if !reflect.DeepEqual(expectedHeaders, frame.Headers) {
		t.Errorf("Frame 1 should have correct headers")
	}

	expectedBody := []byte{}
	if !bytes.Equal(expectedBody, frame.Body) {
		t.Errorf("Frame 1 should have no body")
	}

	frame, err = parser.NextFrame()

	if err != nil {
		t.Errorf("No error should be raised for frame 2")
	}

	if frame.Command != parsing.CONNECTED {
		t.Errorf("Frame 2 should have type CONNECTED")
	}

	expectedHeaders = map[string]string{
		"version": "1.2",
	}
	if !reflect.DeepEqual(expectedHeaders, frame.Headers) {
		t.Errorf("Frame 2 should have correct headers")
	}

	expectedBody = []byte{}
	if !bytes.Equal(expectedBody, frame.Body) {
		t.Errorf("Frame 2 should have no body")
	}
}

// Mock representation of incoming tcp connection
type mockTCPStream struct {
	streamData  string
	currentByte int
	readNo      int
}

func (mock *mockTCPStream) Read(p []byte) (n int, err error) {
	readSize := READ_SIZES_BYTES[mock.readNo%len(READ_SIZES_BYTES)]
	bytesLeft := len(mock.streamData) - mock.currentByte
	bytesToRead := min(readSize, bytesLeft)

	streamStart := mock.currentByte
	streamEnd := mock.currentByte + bytesToRead

	copy(p, mock.streamData[streamStart:streamEnd])

	mock.readNo += 1
	mock.currentByte += bytesToRead

	n = bytesToRead
	if mock.currentByte == len(mock.streamData) {
		err = io.EOF
	} else {
		err = nil
	}

	return
}

func min(x, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}
