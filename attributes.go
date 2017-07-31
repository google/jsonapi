package jsonapi

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// NOTE: reciever for MarshalJSON() should not be a pointer
// https://play.golang.org/p/Cf9yYLIzJA (MarshalJSON() w/ pointer reciever)
// https://play.golang.org/p/5EsItAtgXy (MarshalJSON() w/o pointer reciever)

const iso8601Layout = "2006-01-02T15:04:05Z07:00"

var (
	jsonUnmarshaler = reflect.TypeOf(new(json.Unmarshaler)).Elem()
)

// iso8601Datetime represents a ISO8601 formatted datetime
// It is a time.Time instance that marshals and unmarshals to the ISO8601 ref
type iso8601Datetime struct {
	time.Time
}

// MarshalJSON implements the json.Marshaler interface.
func (t iso8601Datetime) MarshalJSON() ([]byte, error) {
	s := t.Time.Format(iso8601Layout)
	return json.Marshal(s)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *iso8601Datetime) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return nil
	}
	// Fractional seconds are handled implicitly by Parse.
	var err error
	if t.Time, err = time.Parse(strconv.Quote(iso8601Layout), string(data)); err != nil {
		return ErrInvalidISO8601
	}
	return err
}

// iso8601Datetime.String() - override default String() on time
func (t iso8601Datetime) String() string {
	return t.Format(iso8601Layout)
}

// unix(Unix Seconds) marshals/unmarshals the number of milliseconds elapsed since January 1, 1970 UTC
type unix struct {
	time.Time
}

// MarshalJSON implements the json.Marshaler interface.
func (t unix) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Unix())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *unix) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	s := string(data)
	if s == "null" {
		return nil
	}

	v, err := stringToInt64(s)
	if err != nil {
		// return this specific error to maintain existing tests.
		// TODO: consider refactoring tests to not assert against error string
		return ErrInvalidTime
	}

	t.Time = time.Unix(v, 0).In(time.UTC)

	return nil
}

// unixMilli (Unix Millisecond) marshals/unmarshals the number of milliseconds elapsed since January 1, 1970 UTC
type unixMilli struct {
	time.Time
}

// MarshalJSON implements the json.Marshaler interface.
func (t unixMilli) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.UnixNano() / int64(time.Millisecond))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *unixMilli) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	s := string(data)
	if s == "null" {
		return nil
	}

	v, err := stringToInt64(s)
	if err != nil {
		return err
	}

	t.Time = time.Unix(v/1000, (v % 1000 * int64(time.Millisecond))).In(time.UTC)

	return nil
}

// stringToInt64 convert time in either decimal or exponential notation to int64
// https://golang.org/doc/go1.8#encoding_json
// go1.8 prefers decimal notation
// go1.7 may use exponetial notation, so check if it came in as a float
func stringToInt64(s string) (int64, error) {
	var v int64
	if strings.Contains(s, ".") {
		fv, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return v, err
		}
		v = int64(fv)
	} else {
		iv, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return v, err
		}
		v = iv
	}
	return v, nil
}

func implementsJSONUnmarshaler(t reflect.Type) bool {
	ok, _ := deepCheckImplementation(t, jsonUnmarshaler)
	return ok
}

func deepCheckImplementation(t, interfaceType reflect.Type) (bool, reflect.Type) {
	// check as-is
	if t.Implements(interfaceType) {
		return true, t
	}

	switch t.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice:
		// check ptr implementation
		ptrType := reflect.PtrTo(t)
		if ptrType.Implements(interfaceType) {
			return true, ptrType
		}
		// since these are reference types, re-check on the element of t
		return deepCheckImplementation(t.Elem(), interfaceType)
	default:
		// check ptr implementation
		ptrType := reflect.PtrTo(t)
		if ptrType.Implements(interfaceType) {
			return true, ptrType
		}
		// nothing else to check, return false
		return false, nil
	}
}
