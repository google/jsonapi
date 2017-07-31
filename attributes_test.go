package jsonapi

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestIso8601Datetime(t *testing.T) {
	pacific, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}

	type test struct {
		stringVal string
		dtVal     iso8601Datetime
	}

	tests := []*test{
		&test{
			stringVal: strconv.Quote("2017-04-06T13:00:00-07:00"),
			dtVal:     iso8601Datetime{Time: time.Date(2017, time.April, 6, 13, 0, 0, 0, pacific)},
		},
		&test{
			stringVal: strconv.Quote("2007-05-06T13:00:00-07:00"),
			dtVal:     iso8601Datetime{Time: time.Date(2007, time.May, 6, 13, 0, 0, 0, pacific)},
		},
		&test{
			stringVal: strconv.Quote("2016-12-08T15:18:54Z"),
			dtVal:     iso8601Datetime{Time: time.Date(2016, time.December, 8, 15, 18, 54, 0, time.UTC)},
		},
	}

	for _, test := range tests {
		// unmarshal stringVal by calling UnmarshalJSON()
		dt := &iso8601Datetime{}
		if err := dt.UnmarshalJSON([]byte(test.stringVal)); err != nil {
			t.Fatal(err)
		}

		// compare unmarshaled stringVal to dtVal
		if !dt.Time.Equal(test.dtVal.Time) {
			t.Errorf("\n\tE=%+v\n\tA=%+v", test.dtVal.UnixNano(), dt.UnixNano())
		}

		// marshal dtVal by calling MarshalJSON()
		b, err := test.dtVal.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}

		// compare marshaled dtVal to stringVal
		if test.stringVal != string(b) {
			t.Errorf("\n\tE=%+v\n\tA=%+v", test.stringVal, string(b))
		}
	}
}

func TestUnixMilliVariations(t *testing.T) {
	control := unixMilli{
		Time: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
	}

	{
		var val map[string]unixMilli
		t.Logf("\nval: %#v\n", val)

		payload := []byte(`{"foo": 1257894000000, "bar":1257894000000}`)
		json.Unmarshal(payload, &val)

		if !val["foo"].Time.Equal(control.Time) {
			t.Errorf("\n\tE=%+v\n\tA=%+v", control.Time, val["foo"].Time)
		}

		b, _ := json.Marshal(val)
		is, err := isJSONEqual(b, payload)
		if err != nil {
			t.Fatal(err)
		}
		if !is {
			t.Errorf("\n\tE=%s\n\tA=%s", payload, b)
		}
	}
	{
		var val map[string]*unixMilli
		t.Logf("\nval: %#v\n", val)

		payload := []byte(`{"foo": 1257894000000, "bar":1257894000000}`)
		json.Unmarshal(payload, &val)

		if !val["foo"].Time.Equal(control.Time) {
			t.Errorf("\n\tE=%+v\n\tA=%+v", control.Time, val["foo"].Time)
		}

		b, _ := json.Marshal(val)
		is, err := isJSONEqual(b, payload)
		if err != nil {
			t.Fatal(err)
		}
		if !is {
			t.Errorf("\n\tE=%s\n\tA=%s", payload, b)
		}
	}
	{
		var val []*unixMilli
		t.Logf("\nval: %#v\n", val)

		payload := []byte(`[1257894000000,1257894000000]`)
		json.Unmarshal(payload, &val)

		if !val[0].Time.Equal(control.Time) {
			t.Errorf("\n\tE=%+v\n\tA=%+v", control.Time, val[0].Time)
		}

		b, _ := json.Marshal(val)
		is, err := isJSONEqual(b, payload)
		if err != nil {
			t.Fatal(err)
		}
		if !is {
			t.Errorf("\n\tE=%s\n\tA=%s", payload, b)
		}
	}
	{
		var val []unixMilli
		t.Logf("\nval: %#v\n", val)

		payload := []byte(`[1257894000000,1257894000000]`)
		json.Unmarshal(payload, &val)

		if !val[0].Time.Equal(control.Time) {
			t.Errorf("\n\tE=%+v\n\tA=%+v", control.Time, val[0].Time)
		}

		b, _ := json.Marshal(val)
		is, err := isJSONEqual(b, payload)
		if err != nil {
			t.Fatal(err)
		}
		if !is {
			t.Errorf("\n\tE=%s\n\tA=%s", payload, b)
		}
	}
	{
		var val unixMilli
		t.Logf("\nval: %#v\n", val)

		payload := []byte(`1257894000000`)
		json.Unmarshal(payload, &val)

		if !val.Time.Equal(control.Time) {
			t.Errorf("\n\tE=%+v\n\tA=%+v", control.Time, val.Time)
		}

		b, _ := json.Marshal(val)
		is, err := isJSONEqual(b, payload)
		if err != nil {
			t.Fatal(err)
		}
		if !is {
			t.Errorf("\n\tE=%s\n\tA=%s", payload, b)
		}
	}
	{
		var val *unixMilli
		t.Logf("\nval: %#v\n", val)

		payload := []byte(`1257894000000`)
		json.Unmarshal(payload, &val)

		if !val.Time.Equal(control.Time) {
			t.Errorf("\n\tE=%+v\n\tA=%+v", control.Time, val.Time)
		}

		b, _ := json.Marshal(val)
		is, err := isJSONEqual(b, payload)
		if err != nil {
			t.Fatal(err)
		}
		if !is {
			t.Errorf("\n\tE=%s\n\tA=%s", payload, b)
		}
	}
}
func TestUnixMilli(t *testing.T) {
	type test struct {
		stringVal string
		dtVal     unixMilli
	}

	tests := []*test{
		&test{
			stringVal: "1257894000000",
			dtVal:     unixMilli{Time: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
		},
		&test{
			stringVal: "1257894000999",
			dtVal:     unixMilli{Time: time.Date(2009, time.November, 10, 23, 0, 0, 999000000, time.UTC)},
		},
	}

	for _, test := range tests {
		// unmarshal stringVal by calling UnmarshalJSON()
		dt := &unixMilli{}
		if err := dt.UnmarshalJSON([]byte(test.stringVal)); err != nil {
			t.Fatal(err)
		}

		// compare unmarshaled stringVal to dtVal
		if !dt.Time.Equal(test.dtVal.Time) {
			t.Errorf("\n\tE=%+v\n\tA=%+v", test.dtVal.UnixNano(), dt.UnixNano())
		}

		// marshal dtVal by calling MarshalJSON()
		b, err := test.dtVal.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}

		// compare marshaled dtVal to stringVal
		if test.stringVal != string(b) {
			t.Errorf("\n\tE=%+v\n\tA=%+v", test.stringVal, string(b))
		}
	}
}

func TestImplementsJSONUnmarshaler(t *testing.T) {
	{ // positive
		raw := json.RawMessage{}
		typ := reflect.TypeOf(&raw)
		if ok := implementsJSONUnmarshaler(typ); !ok {
			t.Error("expected json.RawMessage to implement json.Unmarshaler")
		}
	}
	{ // positive
		isoDateTime := iso8601Datetime{}
		typ := reflect.TypeOf(&isoDateTime)
		if ok := implementsJSONUnmarshaler(typ); !ok {
			t.Error("expected ISO8601Datetime to implement json.Unmarshaler")
		}
	}
	{ // negative
		type customString string
		input := customString("foo")
		typ := reflect.TypeOf(&input)
		if ok := implementsJSONUnmarshaler(typ); ok {
			t.Error("got true; expected customString to not implement json.Unmarshaler")
		}
	}
}

func TestDeepCheckImplementation(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "concrete ( RawMessage is a reflect.Type of slice)",
			input: json.RawMessage{},
		},
		{
			name:  "RawMessage ptr",
			input: &json.RawMessage{},
		},
		{
			name:  "concrete slice of RawMessage",
			input: []json.RawMessage{},
		},
		{
			name:  "slice of RawMessage ptrs",
			input: []*json.RawMessage{},
		},
		{
			name:  "concrete map of RawMessage",
			input: map[string]json.RawMessage{},
		},
		{
			name:  "map of RawMessage ptrs",
			input: map[string]*json.RawMessage{},
		},
		{
			name:  "map of RawMessage slice",
			input: map[string][]json.RawMessage{},
		},
		{
			name: "ptr ptr of RawMessage",
			input: func() **json.RawMessage {
				r := &json.RawMessage{}
				return &r
			}(),
		},
		{
			name:  "concrete unixMilli (struct)",
			input: unixMilli{},
		},
		{
			name:  "unixMilli ptr",
			input: &unixMilli{},
		},
		{
			name:  "concrete slice of unixMilli",
			input: []unixMilli{},
		},
		{
			name:  "slice of unixMilli ptrs",
			input: []*unixMilli{},
		},
		{
			name:  "concrete map of unixMilli",
			input: map[string]unixMilli{},
		},
		{
			name:  "map of unixMilli ptrs",
			input: map[string]*unixMilli{},
		},
		{
			name:  "map of unixMilli slice",
			input: map[string][]unixMilli{},
		},
		{
			name: "ptr ptr of unixMilli",
			input: func() **unixMilli {
				r := &unixMilli{}
				return &r
			}(),
		},
	}

	for _, scenario := range tests {
		typ := reflect.TypeOf(scenario.input)
		ok, elemType := deepCheckImplementation(typ, jsonUnmarshaler)
		if !ok {
			t.Errorf("\n\tE=%v\n\tA=%v", typ, elemType)
		}
	}
}
