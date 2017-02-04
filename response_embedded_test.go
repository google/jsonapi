package jsonapi

import (
	"bytes"
	"reflect"
	"testing"
)

func TestMergeNode(t *testing.T) {
	parent := &Node{
		Type:       "Good",
		ID:         "99",
		Attributes: map[string]interface{}{"fizz": "buzz"},
	}

	child := &Node{
		Type:       "Better",
		ClientID:   "1111",
		Attributes: map[string]interface{}{"timbuk": 2},
	}

	expected := &Node{
		Type:       "Better",
		ID:         "99",
		ClientID:   "1111",
		Attributes: map[string]interface{}{"fizz": "buzz", "timbuk": 2},
	}

	parent.merge(child)

	if !reflect.DeepEqual(expected, parent) {
		t.Errorf("Got %+v Expected %+v", parent, expected)
	}
}

func TestIsEmbeddedStruct(t *testing.T) {
	type foo struct{}

	structType := reflect.TypeOf(foo{})
	stringType := reflect.TypeOf("")
	if structType.Kind() != reflect.Struct {
		t.Fatal("structType.Kind() is not a struct.")
	}
	if stringType.Kind() != reflect.String {
		t.Fatal("stringType.Kind() is not a string.")
	}

	type test struct {
		scenario    string
		input       reflect.StructField
		expectedRes bool
	}

	tests := []test{
		test{
			scenario:    "success",
			input:       reflect.StructField{Anonymous: true, Type: structType},
			expectedRes: true,
		},
		test{
			scenario:    "wrong type",
			input:       reflect.StructField{Anonymous: true, Type: stringType},
			expectedRes: false,
		},
		test{
			scenario:    "not embedded",
			input:       reflect.StructField{Type: structType},
			expectedRes: false,
		},
	}

	for _, test := range tests {
		res := isEmbeddedStruct(test.input)
		if res != test.expectedRes {
			t.Errorf("Scenario -> %s\nGot -> %v\nExpected -> %v\n", test.scenario, res, test.expectedRes)
		}
	}
}

func TestShouldIgnoreField(t *testing.T) {
	type test struct {
		scenario    string
		input       string
		expectedRes bool
	}

	tests := []test{
		test{
			scenario:    "opt-out",
			input:       annotationIgnore,
			expectedRes: true,
		},
		test{
			scenario:    "no tag",
			input:       "",
			expectedRes: false,
		},
		test{
			scenario:    "wrong tag",
			input:       "wrong,tag",
			expectedRes: false,
		},
	}

	for _, test := range tests {
		res := shouldIgnoreField(test.input)
		if res != test.expectedRes {
			t.Errorf("Scenario -> %s\nGot -> %v\nExpected -> %v\n", test.scenario, res, test.expectedRes)
		}
	}
}

func TestIsValidEmbeddedStruct(t *testing.T) {
	type foo struct{}

	structType := reflect.TypeOf(foo{})
	stringType := reflect.TypeOf("")
	if structType.Kind() != reflect.Struct {
		t.Fatal("structType.Kind() is not a struct.")
	}
	if stringType.Kind() != reflect.String {
		t.Fatal("stringType.Kind() is not a string.")
	}

	type test struct {
		scenario    string
		input       reflect.StructField
		expectedRes bool
	}

	tests := []test{
		test{
			scenario:    "success",
			input:       reflect.StructField{Anonymous: true, Type: structType},
			expectedRes: true,
		},
		test{
			scenario:    "opt-out",
			input:       reflect.StructField{Anonymous: true, Tag: "jsonapi:\"-\"", Type: structType},
			expectedRes: false,
		},
		test{
			scenario:    "wrong type",
			input:       reflect.StructField{Anonymous: true, Type: stringType},
			expectedRes: false,
		},
		test{
			scenario:    "not embedded",
			input:       reflect.StructField{Type: structType},
			expectedRes: false,
		},
	}

	for _, test := range tests {
		res := (isEmbeddedStruct(test.input) && !shouldIgnoreField(test.input.Tag.Get(annotationJSONAPI)))
		if res != test.expectedRes {
			t.Errorf("Scenario -> %s\nGot -> %v\nExpected -> %v\n", test.scenario, res, test.expectedRes)
		}
	}
}

func TestMarshalUnmarshalCompositeStruct(t *testing.T) {
	type Thing struct {
		ID   int    `jsonapi:"primary,things"`
		Fizz string `jsonapi:"attr,fizz"`
		Buzz int    `jsonapi:"attr,buzz"`
	}

	type Model struct {
		Thing
		Foo string `jsonapi:"attr,foo"`
		Bar string `jsonapi:"attr,bar"`
		Bat string `jsonapi:"attr,bat"`
	}

	model := &Model{}
	model.ID = 1
	model.Fizz = "fizzy"
	model.Buzz = 99
	model.Foo = "fooey"
	model.Bar = "barry"
	model.Bat = "batty"

	buf := bytes.NewBuffer(nil)
	if err := MarshalOnePayload(buf, model); err != nil {
		t.Fatal(err)
	}

	// TODO: redo this
	// assert encoding from model to jsonapi output
	// expected := `{"data":{"type":"things","id":"1","attributes":{"bar":"barry","bat":"batty","buzz":99,"fizz":"fizzy","foo":"fooey"}}}`
	// if expected != string(buf.Bytes()) {
	// 	t.Errorf("Got %+v Expected %+v", string(buf.Bytes()), expected)
	// }

	dst := &Model{}
	if err := UnmarshalPayload(buf, dst); err != nil {
		t.Fatal(err)
	}

	// assert decoding from jsonapi output to model
	if !reflect.DeepEqual(model, dst) {
		t.Errorf("Got %#v Expected %#v", dst, model)
	}
}
