package jsonapi

import (
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestISO8601Datetime(t *testing.T) {
	pacific, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}

	type test struct {
		stringVal string
		dtVal     ISO8601Datetime
	}

	tests := []*test{
		&test{
			stringVal: strconv.Quote("2017-04-06T13:00:00-07:00"),
			dtVal:     ISO8601Datetime{Time: time.Date(2017, time.April, 6, 13, 0, 0, 0, pacific)},
		},
		&test{
			stringVal: strconv.Quote("2007-05-06T13:00:00-07:00"),
			dtVal:     ISO8601Datetime{Time: time.Date(2007, time.May, 6, 13, 0, 0, 0, pacific)},
		},
		&test{
			stringVal: strconv.Quote("2016-12-08T15:18:54Z"),
			dtVal:     ISO8601Datetime{Time: time.Date(2016, time.December, 8, 15, 18, 54, 0, time.UTC)},
		},
	}

	for _, test := range tests {
		// unmarshal stringVal by calling UnmarshalJSON()
		dt := &ISO8601Datetime{}
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

func TestUnixMilli(t *testing.T) {
	type test struct {
		stringVal string
		dtVal     UnixMilli
	}

	tests := []*test{
		&test{
			stringVal: "1257894000000",
			dtVal:     UnixMilli{Time: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
		},
		&test{
			stringVal: "1257894000999",
			dtVal:     UnixMilli{Time: time.Date(2009, time.November, 10, 23, 0, 0, 999000000, time.UTC)},
		},
	}

	for _, test := range tests {
		// unmarshal stringVal by calling UnmarshalJSON()
		dt := &UnixMilli{}
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

func TestIsJSONMarshaler(t *testing.T) {
	{ // positive
		isoDateTime := ISO8601Datetime{}
		v := reflect.ValueOf(&isoDateTime)
		if _, ok := isJSONMarshaler(v); !ok {
			t.Error("got false; expected ISO8601Datetime to implement json.Marshaler")
		}
	}
	{ // negative
		type customString string
		input := customString("foo")
		v := reflect.ValueOf(&input)
		if _, ok := isJSONMarshaler(v); ok {
			t.Error("got true; expected customString to not implement json.Marshaler")
		}
	}
}

func TestIsJSONUnmarshaler(t *testing.T) {
	{ // positive
		isoDateTime := ISO8601Datetime{}
		v := reflect.ValueOf(&isoDateTime)
		if _, ok := isJSONUnmarshaler(v); !ok {
			t.Error("expected ISO8601Datetime to implement json.Unmarshaler")
		}
	}
	{ // negative
		type customString string
		input := customString("foo")
		v := reflect.ValueOf(&input)
		if _, ok := isJSONUnmarshaler(v); ok {
			t.Error("got true; expected customString to not implement json.Unmarshaler")
		}
	}
}
