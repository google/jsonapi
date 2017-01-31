package jsonapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"testing"
)

func TestErrorObjectWritesExpectedErrorMessage(t *testing.T) {
	err := &ErrorObject{Title: "Title test.", Detail: "Detail test."}
	var input error = err

	output := input.Error()

	if output != fmt.Sprintf("Error: %s %s\n", err.Title, err.Detail) {
		t.Fatal("Unexpected output.")
	}
}

func TestMarshalErrorsWritesTheExpectedPayload(t *testing.T) {
	var marshalErrorsTableTasts = []struct {
		In  []*ErrorObject
		Out map[string]interface{}
	}{
		{ // This tests that given fields are turned into the appropriate JSON representation.
			In: []*ErrorObject{{ID: "0", Title: "Test title.", Detail: "Test detail", Status: "400", Code: "E1100"}},
			Out: map[string]interface{}{"errors": []interface{}{
				map[string]interface{}{"id": "0", "title": "Test title.", "detail": "Test detail", "status": "400", "code": "E1100"},
			}},
		},
		{ // This tests that the `Meta` field is serialized properly.
			In: []*ErrorObject{{Title: "Test title.", Detail: "Test detail", Meta: &map[string]interface{}{"key": "val"}}},
			Out: map[string]interface{}{"errors": []interface{}{
				map[string]interface{}{"title": "Test title.", "detail": "Test detail", "meta": map[string]interface{}{"key": "val"}},
			}},
		},
	}
	for _, testRow := range marshalErrorsTableTasts {
		buffer, output := bytes.NewBuffer(nil), map[string]interface{}{}
		var writer io.Writer = buffer

		_ = MarshalErrors(writer, testRow.In)
		json.Unmarshal(buffer.Bytes(), &output)

		if !reflect.DeepEqual(output, testRow.Out) {
			t.Fatalf("Expected: \n%#v \nto equal: \n%#v", output, testRow.Out)
		}
	}
}
