package jsonapi

import (
	"bytes"
	"encoding/json"
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

// TestEmbeddedUnmarshalOrder tests the behavior of the marshaler/unmarshaler of embedded structs
// when a struct has an embedded struct w/ competing attributes, the top-level attributes take precedence
// it compares the behavior against the standard json package
func TestEmbeddedUnmarshalOrder(t *testing.T) {
	type Bar struct {
		Name int `jsonapi:"attr,Name"`
	}

	type Foo struct {
		Bar
		ID   string `jsonapi:"primary,foos"`
		Name string `jsonapi:"attr,Name"`
	}

	f := &Foo{
		ID:   "1",
		Name: "foo",
		Bar: Bar{
			Name: 5,
		},
	}

	// marshal f (Foo) using jsonapi marshaler
	jsonAPIData := bytes.NewBuffer(nil)
	if err := MarshalPayload(jsonAPIData, f); err != nil {
		t.Fatal(err)
	}

	// marshal f (Foo) using json marshaler
	jsonData, err := json.Marshal(f)

	// convert bytes to map[string]interface{} so that we can do a semantic JSON comparison
	var jsonAPIVal, jsonVal map[string]interface{}
	if err := json.Unmarshal(jsonAPIData.Bytes(), &jsonAPIVal); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(jsonData, &jsonVal); err != nil {
		t.Fatal(err)
	}

	// get to the jsonapi attribute map
	jAttrMap := jsonAPIVal["data"].(map[string]interface{})["attributes"].(map[string]interface{})

	// compare
	if !reflect.DeepEqual(jAttrMap["Name"], jsonVal["Name"]) {
		t.Errorf("Got\n%s\nExpected\n%s\n", jAttrMap["Name"], jsonVal["Name"])
	}
}

// TestEmbeddedMarshalOrder tests the behavior of the marshaler/unmarshaler of embedded structs
// when a struct has an embedded struct w/ competing attributes, the top-level attributes take precedence
// it compares the behavior against the standard json package
func TestEmbeddedMarshalOrder(t *testing.T) {
	type Bar struct {
		Name int `jsonapi:"attr,Name"`
	}

	type Foo struct {
		Bar
		ID   string `jsonapi:"primary,foos"`
		Name string `jsonapi:"attr,Name"`
	}

	// get a jsonapi payload w/ Name attribute of an int type
	payloadWithInt, err := json.Marshal(&OnePayload{
		Data: &Node{
			Type: "foos",
			ID:   "1",
			Attributes: map[string]interface{}{
				"Name": 5,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// get a jsonapi payload w/ Name attribute of an string type
	payloadWithString, err := json.Marshal(&OnePayload{
		Data: &Node{
			Type: "foos",
			ID:   "1",
			Attributes: map[string]interface{}{
				"Name": "foo",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// unmarshal payloadWithInt to f (Foo) using jsonapi unmarshaler; expecting an error
	f := &Foo{}
	if err := UnmarshalPayload(bytes.NewReader(payloadWithInt), f); err == nil {
		t.Errorf("expected an error: int value of 5 should attempt to map to Foo.Name (string) and error")
	}

	// unmarshal payloadWithString to f (Foo) using jsonapi unmarshaler; expecting no error
	f = &Foo{}
	if err := UnmarshalPayload(bytes.NewReader(payloadWithString), f); err != nil {
		t.Error(err)
	}
	if f.Name != "foo" {
		t.Errorf("Got\n%s\nExpected\n%s\n", "foo", f.Name)
	}

	// get a json payload w/ Name attribute of an int type
	bWithInt, err := json.Marshal(map[string]interface{}{
		"Name": 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	// get a json payload w/ Name attribute of an string type
	bWithString, err := json.Marshal(map[string]interface{}{
		"Name": "foo",
	})
	if err != nil {
		t.Fatal(err)
	}

	// unmarshal bWithInt to f (Foo) using json unmarshaler; expecting an error
	f = &Foo{}
	if err := json.Unmarshal(bWithInt, f); err == nil {
		t.Errorf("expected an error: int value of 5 should attempt to map to Foo.Name (string) and error")
	}
	// unmarshal bWithString to f (Foo) using json unmarshaler; expecting no error
	f = &Foo{}
	if err := json.Unmarshal(bWithString, f); err != nil {
		t.Error(err)
	}
	if f.Name != "foo" {
		t.Errorf("Got\n%s\nExpected\n%s\n", "foo", f.Name)
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

	type test struct {
		name          string
		payload       *OnePayload
		dst, expected interface{}
	}

	scenarios := []test{}

	scenarios = append(scenarios, test{
		name: "Model embeds Thing, models have no annotation overlaps",
		dst:  &Model{},
		payload: &OnePayload{
			Data: &Node{
				Type: "things",
				ID:   "1",
				Attributes: map[string]interface{}{
					"bar":  "barry",
					"bat":  "batty",
					"buzz": 99,
					"fizz": "fizzy",
					"foo":  "fooey",
				},
			},
		},
		expected: &Model{
			Foo: "fooey",
			Bar: "barry",
			Bat: "batty",
			Thing: Thing{
				ID:   1,
				Fizz: "fizzy",
				Buzz: 99,
			},
		},
	})

	{
		type Model struct {
			Thing
			Foo  string `jsonapi:"attr,foo"`
			Bar  string `jsonapi:"attr,bar"`
			Bat  string `jsonapi:"attr,bat"`
			Buzz int    `jsonapi:"attr,buzz"` // overrides Thing.Buzz
		}

		scenarios = append(scenarios, test{
			name: "Model embeds Thing, overlap Buzz attribute",
			dst:  &Model{},
			payload: &OnePayload{
				Data: &Node{
					Type: "things",
					ID:   "1",
					Attributes: map[string]interface{}{
						"bar":  "barry",
						"bat":  "batty",
						"buzz": 99,
						"fizz": "fizzy",
						"foo":  "fooey",
					},
				},
			},
			expected: &Model{
				Foo:  "fooey",
				Bar:  "barry",
				Bat:  "batty",
				Buzz: 99,
				Thing: Thing{
					ID:   1,
					Fizz: "fizzy",
				},
			},
		})
	}

	{
		type Model struct {
			Thing
			ModelID int    `jsonapi:"primary,models"` //overrides Thing.ID due to primary annotation
			Foo     string `jsonapi:"attr,foo"`
			Bar     string `jsonapi:"attr,bar"`
			Bat     string `jsonapi:"attr,bat"`
			Buzz    int    `jsonapi:"attr,buzz"` // overrides Thing.Buzz
		}

		scenarios = append(scenarios, test{
			name: "Model embeds Thing, attribute, and primary annotation overlap",
			dst:  &Model{},
			payload: &OnePayload{
				Data: &Node{
					Type: "models",
					ID:   "1",
					Attributes: map[string]interface{}{
						"bar":  "barry",
						"bat":  "batty",
						"buzz": 99,
						"fizz": "fizzy",
						"foo":  "fooey",
					},
				},
			},
			expected: &Model{
				ModelID: 1,
				Foo:     "fooey",
				Bar:     "barry",
				Bat:     "batty",
				Buzz:    99,
				Thing: Thing{
					Fizz: "fizzy",
				},
			},
		})
	}

	{
		type Model struct {
			Thing   `jsonapi:"-"`
			ModelID int    `jsonapi:"primary,models"`
			Foo     string `jsonapi:"attr,foo"`
			Bar     string `jsonapi:"attr,bar"`
			Bat     string `jsonapi:"attr,bat"`
			Buzz    int    `jsonapi:"attr,buzz"`
		}

		scenarios = append(scenarios, test{
			name: "Model embeds Thing, but is annotated w/ ignore",
			dst:  &Model{},
			payload: &OnePayload{
				Data: &Node{
					Type: "models",
					ID:   "1",
					Attributes: map[string]interface{}{
						"bar":  "barry",
						"bat":  "batty",
						"buzz": 99,
						"foo":  "fooey",
					},
				},
			},
			expected: &Model{
				ModelID: 1,
				Foo:     "fooey",
				Bar:     "barry",
				Bat:     "batty",
				Buzz:    99,
			},
		})
	}
	{
		type Model struct {
			*Thing
			ModelID int    `jsonapi:"primary,models"`
			Foo     string `jsonapi:"attr,foo"`
			Bar     string `jsonapi:"attr,bar"`
			Bat     string `jsonapi:"attr,bat"`
		}

		scenarios = append(scenarios, test{
			name: "Model embeds pointer of Thing; Thing is initialized in advance",
			dst:  &Model{Thing: &Thing{}},
			payload: &OnePayload{
				Data: &Node{
					Type: "models",
					ID:   "1",
					Attributes: map[string]interface{}{
						"bar":  "barry",
						"bat":  "batty",
						"foo":  "fooey",
						"buzz": 99,
						"fizz": "fizzy",
					},
				},
			},
			expected: &Model{
				Thing: &Thing{
					Fizz: "fizzy",
					Buzz: 99,
				},
				ModelID: 1,
				Foo:     "fooey",
				Bar:     "barry",
				Bat:     "batty",
			},
		})
	}
	{
		type Model struct {
			*Thing
			ModelID int    `jsonapi:"primary,models"`
			Foo     string `jsonapi:"attr,foo"`
			Bar     string `jsonapi:"attr,bar"`
			Bat     string `jsonapi:"attr,bat"`
		}

		scenarios = append(scenarios, test{
			name: "Model embeds pointer of Thing; Thing is initialized w/ Unmarshal",
			dst:  &Model{},
			payload: &OnePayload{
				Data: &Node{
					Type: "models",
					ID:   "1",
					Attributes: map[string]interface{}{
						"bar":  "barry",
						"bat":  "batty",
						"foo":  "fooey",
						"buzz": 99,
						"fizz": "fizzy",
					},
				},
			},
			expected: &Model{
				Thing: &Thing{
					Fizz: "fizzy",
					Buzz: 99,
				},
				ModelID: 1,
				Foo:     "fooey",
				Bar:     "barry",
				Bat:     "batty",
			},
		})
	}
	{
		type Model struct {
			*Thing
			ModelID int    `jsonapi:"primary,models"`
			Foo     string `jsonapi:"attr,foo"`
			Bar     string `jsonapi:"attr,bar"`
			Bat     string `jsonapi:"attr,bat"`
		}

		scenarios = append(scenarios, test{
			name: "Model embeds pointer of Thing; jsonapi model doesn't assign anything to Thing; *Thing is nil",
			dst:  &Model{},
			payload: &OnePayload{
				Data: &Node{
					Type: "models",
					ID:   "1",
					Attributes: map[string]interface{}{
						"bar": "barry",
						"bat": "batty",
						"foo": "fooey",
					},
				},
			},
			expected: &Model{
				ModelID: 1,
				Foo:     "fooey",
				Bar:     "barry",
				Bat:     "batty",
			},
		})
	}

	{
		type Model struct {
			*Thing
			ModelID int    `jsonapi:"primary,models"`
			Foo     string `jsonapi:"attr,foo"`
			Bar     string `jsonapi:"attr,bar"`
			Bat     string `jsonapi:"attr,bat"`
		}

		scenarios = append(scenarios, test{
			name: "Model embeds pointer of Thing; *Thing is nil",
			dst:  &Model{},
			payload: &OnePayload{
				Data: &Node{
					Type: "models",
					ID:   "1",
					Attributes: map[string]interface{}{
						"bar": "barry",
						"bat": "batty",
						"foo": "fooey",
					},
				},
			},
			expected: &Model{
				ModelID: 1,
				Foo:     "fooey",
				Bar:     "barry",
				Bat:     "batty",
			},
		})
	}
	for _, scenario := range scenarios {
		t.Logf("running scenario: %s\n", scenario.name)

		// get the expected model and marshal to jsonapi
		buf := bytes.NewBuffer(nil)
		if err := MarshalPayload(buf, scenario.expected); err != nil {
			t.Fatal(err)
		}

		// get the node model representation and marshal to jsonapi
		payload, err := json.Marshal(scenario.payload)
		if err != nil {
			t.Fatal(err)
		}

		// assert that we're starting w/ the same payload
		isJSONEqual, err := isJSONEqual(payload, buf.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		if !isJSONEqual {
			t.Errorf("Got\n%s\nExpected\n%s\n", buf.Bytes(), payload)
		}

		// run jsonapi unmarshal
		if err := UnmarshalPayload(bytes.NewReader(payload), scenario.dst); err != nil {
			t.Fatal(err)
		}

		// assert decoded and expected models are equal
		if !reflect.DeepEqual(scenario.expected, scenario.dst) {
			t.Errorf("Got\n%#v\nExpected\n%#v\n", scenario.dst, scenario.expected)
		}
	}
}

func TestMarshal_duplicatePrimaryAnnotationFromEmbededStructs(t *testing.T) {
	type Outer struct {
		ID string `jsonapi:"primary,outer"`
		Comment
		*Post
	}

	o := Outer{
		ID:      "outer",
		Comment: Comment{ID: 1},
		Post:    &Post{ID: 5},
	}
	var payloadData map[string]interface{}

	// Test the standard libraries JSON handling of dup (ID) fields
	jsonData, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(jsonData, &payloadData); err != nil {
		t.Fatal(err)
	}
	if e, a := o.ID, payloadData["ID"]; e != a {
		t.Fatalf("Was expecting ID to be %v, got %v", e, a)
	}

	// Test the JSONAPI lib handling of dup (ID) fields
	jsonAPIData := new(bytes.Buffer)
	if err := MarshalPayload(jsonAPIData, &o); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(jsonAPIData.Bytes(), &payloadData); err != nil {
		t.Fatal(err)
	}
	data := payloadData["data"].(map[string]interface{})
	id := data["id"].(string)
	if e, a := o.ID, id; e != a {
		t.Fatalf("Was expecting ID to be %v, got %v", e, a)
	}
}

func TestMarshal_duplicateAttributeAnnotationFromEmbededStructs(t *testing.T) {
	type Foo struct {
		Count uint `json:"count" jsonapi:"attr,count"`
	}
	type Bar struct {
		Count uint `json:"count" jsonapi:"attr,count"`
	}
	type Outer struct {
		ID uint `json:"id" jsonapi:"primary,outer"`
		Foo
		Bar
	}
	o := Outer{
		ID:  1,
		Foo: Foo{Count: 1},
		Bar: Bar{Count: 2},
	}

	var payloadData map[string]interface{}

	// The standard JSON lib will not serialize either embeded struct's fields if
	// a duplicate is encountered
	jsonData, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(jsonData, &payloadData); err != nil {
		t.Fatal(err)
	}
	if _, found := payloadData["count"]; found {
		t.Fatalf("Was not expecting to find the `count` key in the JSON")
	}

	// Test the JSONAPI lib handling of dup (attr) fields
	jsonAPIData := new(bytes.Buffer)
	if err := MarshalPayload(jsonAPIData, &o); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(jsonAPIData.Bytes(), &payloadData); err != nil {
		t.Fatal(err)
	}
	data := payloadData["data"].(map[string]interface{})
	if _, found := data["attributes"]; found {
		t.Fatal("Was not expecting to find any `attributes` in the JSON API")
	}
}

func TestMarshal_duplicateAttributeAnnotationFromEmbededStructsPtrs(t *testing.T) {
	type Foo struct {
		Count uint `json:"count" jsonapi:"attr,count"`
	}
	type Bar struct {
		Count uint `json:"count" jsonapi:"attr,count"`
	}
	type Outer struct {
		ID uint `json:"id" jsonapi:"primary,outer"`
		*Foo
		*Bar
	}
	o := Outer{
		ID:  1,
		Foo: &Foo{Count: 1},
		Bar: &Bar{Count: 2},
	}

	var payloadData map[string]interface{}

	// The standard JSON lib will not serialize either embeded struct's fields if
	// a duplicate is encountered
	jsonData, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(jsonData, &payloadData); err != nil {
		t.Fatal(err)
	}
	if _, found := payloadData["count"]; found {
		t.Fatalf("Was not expecting to find the `count` key in the JSON")
	}

	// Test the JSONAPI lib handling of dup (attr) fields
	jsonAPIData := new(bytes.Buffer)
	if err := MarshalPayload(jsonAPIData, &o); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(jsonAPIData.Bytes(), &payloadData); err != nil {
		t.Fatal(err)
	}
	data := payloadData["data"].(map[string]interface{})
	if _, found := data["attributes"]; found {
		t.Fatal("Was not expecting to find any `attributes` in the JSON API")
	}
}

func TestMarshal_duplicateAttributeAnnotationFromEmbededStructsMixed(t *testing.T) {
	type Foo struct {
		Count uint `json:"count" jsonapi:"attr,count"`
	}
	type Bar struct {
		Count uint `json:"count" jsonapi:"attr,count"`
	}
	type Outer struct {
		ID uint `json:"id" jsonapi:"primary,outer"`
		*Foo
		Bar
	}
	o := Outer{
		ID:  1,
		Foo: &Foo{Count: 1},
		Bar: Bar{Count: 2},
	}

	var payloadData map[string]interface{}

	// The standard JSON lib will not serialize either embeded struct's fields if
	// a duplicate is encountered
	jsonData, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(jsonData, &payloadData); err != nil {
		t.Fatal(err)
	}
	if _, found := payloadData["count"]; found {
		t.Fatalf("Was not expecting to find the `count` key in the JSON")
	}

	// Test the JSONAPI lib handling of dup (attr) fields; it should serialize
	// neither
	jsonAPIData := new(bytes.Buffer)
	if err := MarshalPayload(jsonAPIData, &o); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(jsonAPIData.Bytes(), &payloadData); err != nil {
		t.Fatal(err)
	}
	data := payloadData["data"].(map[string]interface{})
	if _, found := data["attributes"]; found {
		t.Fatal("Was not expecting to find any `attributes` in the JSON API")
	}
}

func TestMarshal_duplicateFieldFromEmbededStructs_serializationNameDiffers(t *testing.T) {
	type Foo struct {
		Count uint `json:"foo-count" jsonapi:"attr,foo-count"`
	}
	type Bar struct {
		Count uint `json:"bar-count" jsonapi:"attr,bar-count"`
	}
	type Outer struct {
		ID uint `json:"id" jsonapi:"primary,outer"`
		Foo
		Bar
	}
	o := Outer{
		ID:  1,
		Foo: Foo{Count: 1},
		Bar: Bar{Count: 2},
	}

	var payloadData map[string]interface{}

	// The standard JSON lib will both the fields since their annotation name
	// differs
	jsonData, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(jsonData, &payloadData); err != nil {
		t.Fatal(err)
	}
	fooJSON, fooFound := payloadData["foo-count"]
	if !fooFound {
		t.Fatal("Was expecting to find the `foo-count` key in the JSON")
	}
	if e, a := o.Foo.Count, fooJSON.(float64); e != uint(a) {
		t.Fatalf("Was expecting the `foo-count` value to be %v, got %v", e, a)
	}
	barJSON, barFound := payloadData["bar-count"]
	if !barFound {
		t.Fatal("Was expecting to find the `bar-count` key in the JSON")
	}
	if e, a := o.Bar.Count, barJSON.(float64); e != uint(a) {
		t.Fatalf("Was expecting the `bar-count` value to be %v, got %v", e, a)
	}

	// Test the JSONAPI lib handling; it should serialize both
	jsonAPIData := new(bytes.Buffer)
	if err := MarshalPayload(jsonAPIData, &o); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(jsonAPIData.Bytes(), &payloadData); err != nil {
		t.Fatal(err)
	}
	data := payloadData["data"].(map[string]interface{})
	attributes := data["attributes"].(map[string]interface{})
	fooJSONAPI, fooFound := attributes["foo-count"]
	if !fooFound {
		t.Fatal("Was expecting to find the `foo-count` attribute in the JSON API")
	}
	if e, a := o.Foo.Count, fooJSONAPI.(float64); e != uint(e) {
		t.Fatalf("Was expecting the `foo-count` attrobute to be %v, got %v", e, a)
	}
	barJSONAPI, barFound := attributes["bar-count"]
	if !barFound {
		t.Fatal("Was expecting to find the `bar-count` attribute in the JSON API")
	}
	if e, a := o.Bar.Count, barJSONAPI.(float64); e != uint(e) {
		t.Fatalf("Was expecting the `bar-count` attrobute to be %v, got %v", e, a)
	}
}
