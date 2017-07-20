package jsonapi

import (
	"encoding/json"
	"reflect"
	"strconv"
	"time"
)

const iso8601Layout = "2006-01-02T15:04:05Z07:00"

// ISO8601Datetime represents a ISO8601 formatted datetime
// It is a time.Time instance that marshals and unmarshals to the ISO8601 ref
type ISO8601Datetime struct {
	time.Time
}

// MarshalJSON implements the json.Marshaler interface.
func (t *ISO8601Datetime) MarshalJSON() ([]byte, error) {
	s := t.Time.Format(iso8601Layout)
	return json.Marshal(s)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *ISO8601Datetime) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return nil
	}
	// Fractional seconds are handled implicitly by Parse.
	var err error
	t.Time, err = time.Parse(strconv.Quote(iso8601Layout), string(data))
	return err
}

// ISO8601Datetime.String() - override default String() on time
func (t ISO8601Datetime) String() string {
	return t.Format(iso8601Layout)
}

// func to help determine json.Marshaler implementation
// checks both pointer and non-pointer implementations
func isJSONMarshaler(fv reflect.Value) (json.Marshaler, bool) {
	if u, ok := fv.Interface().(json.Marshaler); ok {
		return u, ok
	}

	if !fv.CanAddr() {
		return nil, false
	}

	u, ok := fv.Addr().Interface().(json.Marshaler)
	return u, ok
}

// func to help determine json.Unmarshaler implementation
// checks both pointer and non-pointer implementations
func isJSONUnmarshaler(fv reflect.Value) (json.Unmarshaler, bool) {
	if u, ok := fv.Interface().(json.Unmarshaler); ok {
		return u, ok
	}

	if !fv.CanAddr() {
		return nil, false
	}

	u, ok := fv.Addr().Interface().(json.Unmarshaler)
	return u, ok
}
