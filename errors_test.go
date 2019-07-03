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
		Title string
		In    []*ErrorObject
		Out   map[string]interface{}
	}{
		{
			Title: "TestFieldsAreSerializedAsNeeded",
			In:    []*ErrorObject{{ID: "0", Title: "Test title.", Detail: "Test detail", Status: "400", Code: "E1100"}},
			Out: map[string]interface{}{"errors": []interface{}{
				map[string]interface{}{"id": "0", "title": "Test title.", "detail": "Test detail", "status": "400", "code": "E1100"},
			}},
		},
		{
			Title: "TestMetaFieldIsSerializedProperly",
			In:    []*ErrorObject{{Title: "Test title.", Detail: "Test detail", Meta: &map[string]interface{}{"key": "val"}}},
			Out: map[string]interface{}{"errors": []interface{}{
				map[string]interface{}{"title": "Test title.", "detail": "Test detail", "meta": map[string]interface{}{"key": "val"}},
			}},
		},
	}
	for _, testRow := range marshalErrorsTableTasts {
		t.Run(testRow.Title, func(t *testing.T) {
			buffer, output := bytes.NewBuffer(nil), map[string]interface{}{}
			var writer io.Writer = buffer

			_ = MarshalErrors(writer, testRow.In)
			json.Unmarshal(buffer.Bytes(), &output)

			if !reflect.DeepEqual(output, testRow.Out) {
				t.Fatalf("Expected: \n%#v \nto equal: \n%#v", output, testRow.Out)
			}
		})
	}
}

func TestMarshalErrors(t *testing.T) {
	firstMeta := map[string]interface{}{
		"custom":     "info",
		"custom two": "more info",
	}
	secondMeta := map[string]interface{}{
		"foo": "foo info",
		"bar": "bar info",
	}
	sample := map[string]interface{}{
		"errors": []interface{}{
			map[string]interface{}{
				"id":     "1",
				"title":  "first title",
				"detail": "first detail",
				"status": "400",
				"code":   "first code",
				"meta":   firstMeta,
			},
			map[string]interface{}{
				"id":     "2",
				"title":  "second title",
				"detail": "second detail",
				"status": "404",
				"code":   "second code",
				"meta":   secondMeta,
			},
		},
	}

	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(data)

	errorsPayload, err := UnmarshalErrors(in)
	if err != nil {
		t.Fatal(err)
	}

	expectedPayload := ErrorsPayload{
		Errors: []*ErrorObject{
			{
				ID:     "1",
				Title:  "first title",
				Detail: "first detail",
				Status: "400",
				Code:   "first code",
				Meta:   &firstMeta,
			}, {
				ID:     "2",
				Title:  "second title",
				Detail: "second detail",
				Status: "404",
				Code:   "second code",
				Meta:   &secondMeta,
			},
		},
	}
	if !reflect.DeepEqual(*errorsPayload, expectedPayload) {
		t.Fatalf("Expected: \n%#v \nto equal: \n%#v", errorsPayload, expectedPayload)
	}
}

func TestMarshalErrorsPartialData(t *testing.T) {
	sample := map[string]interface{}{
		"errors": []interface{}{
			map[string]interface{}{
				"status": "400",
			},
			map[string]interface{}{
				"status": "404",
			},
		},
	}

	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(data)

	errorsPayload, err := UnmarshalErrors(in)
	if err != nil {
		t.Fatal(err)
	}

	expectedPayload := ErrorsPayload{Errors: []*ErrorObject{{Status: "400"}, {Status: "404"}}}

	if !reflect.DeepEqual(*errorsPayload, expectedPayload) {
		t.Fatalf("Expected: \n%#v \nto equal: \n%#v", errorsPayload, expectedPayload)
	}
}
