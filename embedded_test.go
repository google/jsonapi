package jsonapi

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

type Inner struct {
	ID      int    `jsonapi:"primary,tests"`
	Content string `jsonapi:"attr,content"`
}

type Outer struct {
	Inner
	Timestamp time.Time `jsonapi:"attr,timestamp,iso8601"`
}

var (
	testUnmarshalString = `{"data": {"type": "tests", "id": "1", "attributes": {"content": "this is a test", "timestamp": "2019-05-17T05:00:00.000Z"}}}`
	testMarshalString   = `{"data":{"type":"tests","id":"3","attributes":{"content":"marshal this","timestamp":"2019-05-22T15:54:15Z"}}}`
	testStruct          = Outer{
		Inner: Inner{
			ID:      1,
			Content: "this is a test",
		},
		Timestamp: time.Now(),
	}
)

func TestEmbeddedUnmarshal(t *testing.T) {
	test := new(Outer)
	err := UnmarshalPayload(strings.NewReader(testUnmarshalString), test)
	if err != nil {
		t.Fatal(err)
	}
	if test.Content != "this is a test" {
		t.Fatalf("expected content of %s received %s", "this is a test", test.Content)
	}
	if test.ID != 1 {
		t.Fatalf("expected an ID of %v received %v", 1, test.ID)
	}
	stamp, err := time.Parse(time.RFC3339, "2019-05-17T05:00:00.000Z")
	if err != nil {
		t.Fatal(err)
	}
	if test.Timestamp != stamp {
		t.Fatalf("expected a Timestamp of %v received %v", t, test.Timestamp)
	}
}

func TestEmbeddedMarshal(t *testing.T) {
	stamp, err := time.Parse(time.RFC3339, "2019-05-22T15:54:15Z")
	test := &Outer{
		Inner: Inner{
			ID:      3,
			Content: "marshal this",
		},
		Timestamp: stamp,
	}
	var b bytes.Buffer
	err = MarshalPayload(&b, test)
	if err != nil {
		t.Fatal(err)
	}
	output := strings.TrimSpace(string(b.Bytes()))
	if output != testMarshalString {
		t.Fatalf("expected: #%v#\nreceived: #%v#", testMarshalString, output)
	}
}
