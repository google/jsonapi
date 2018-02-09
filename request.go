package jsonapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	unsuportedStructTagMsg = "Unsupported jsonapi tag annotation, %s"
)

var (
	// ErrInvalidTime is returned when a struct has a time.Time type field, but
	// the JSON value was not a unix timestamp integer.
	ErrInvalidTime = errors.New("Only numbers can be parsed as dates, unix timestamps")
	// ErrInvalidISO8601 is returned when a struct has a time.Time type field and includes
	// "iso8601" in the tag spec, but the JSON value was not an ISO8601 timestamp string.
	ErrInvalidISO8601 = errors.New("Only strings can be parsed as dates, ISO8601 timestamps")
	// ErrUnknownFieldNumberType is returned when the JSON value was a float
	// (numeric) but the Struct field was a non numeric type (i.e. not int, uint,
	// float, etc)
	ErrUnknownFieldNumberType = errors.New("The struct field was not of a known number type")
	// ErrInvalidBase64Str is returned when an input string is invalid base64
	ErrInvalidBase64Str = errors.New("The input could not be decoded as a base64 string")
	// ErrUnsupportedPtrType is returned when the Struct field was a pointer but
	// the JSON value was of a different type
	ErrUnsupportedPtrType = errors.New("Pointer type in struct is not supported")
	// ErrInvalidType is returned when the given type is incompatible with the expected type.
	ErrInvalidType = errors.New("Invalid type provided") // I wish we used punctuation.
	// ErrUnsupportedSliceType is returned when the given slice type cannot be unmarshaled.
	ErrUnsupportedSliceType = errors.New("Slice type is not supported")
)

// UnmarshalPayload converts an io into a struct instance using jsonapi tags on
// struct fields. This method supports single request payloads only, at the
// moment. Bulk creates and updates are not supported yet.
//
// Will Unmarshal embedded and sideloaded payloads.  The latter is only possible if the
// object graph is complete.  That is, in the "relationships" data there are type and id,
// keys that correspond to records in the "included" array.
//
// For example you could pass it, in, req.Body and, model, a BlogPost
// struct instance to populate in an http handler,
//
//   func CreateBlog(w http.ResponseWriter, r *http.Request) {
//   	blog := new(Blog)
//
//   	if err := jsonapi.UnmarshalPayload(r.Body, blog); err != nil {
//   		http.Error(w, err.Error(), 500)
//   		return
//   	}
//
//   	// ...do stuff with your blog...
//
//   	w.Header().Set("Content-Type", jsonapi.MediaType)
//   	w.WriteHeader(201)
//
//   	if err := jsonapi.MarshalPayload(w, blog); err != nil {
//   		http.Error(w, err.Error(), 500)
//   	}
//   }
//
//
// Visit https://github.com/google/jsonapi#create for more info.
//
// model interface{} should be a pointer to a struct.
func UnmarshalPayload(in io.Reader, model interface{}) error {
	payload := new(OnePayload)

	if err := json.NewDecoder(in).Decode(payload); err != nil {
		return err
	}

	if payload.Included != nil {
		includedMap := make(map[string]*Node)
		for _, included := range payload.Included {
			key := fmt.Sprintf("%s,%s", included.Type, included.ID)
			includedMap[key] = included
		}

		return unmarshalNode(payload.Data, reflect.ValueOf(model), &includedMap)
	}
	return unmarshalNode(payload.Data, reflect.ValueOf(model), nil)
}

// UnmarshalManyPayload converts an io into a set of struct instances using
// jsonapi tags on the type's struct fields.
func UnmarshalManyPayload(in io.Reader, t reflect.Type) ([]interface{}, error) {
	payload := new(ManyPayload)

	if err := json.NewDecoder(in).Decode(payload); err != nil {
		return nil, err
	}

	models := []interface{}{}         // will be populated from the "data"
	includedMap := map[string]*Node{} // will be populate from the "included"

	if payload.Included != nil {
		for _, included := range payload.Included {
			key := fmt.Sprintf("%s,%s", included.Type, included.ID)
			includedMap[key] = included
		}
	}

	for _, data := range payload.Data {
		model := reflect.New(t.Elem())
		err := unmarshalNode(data, model, &includedMap)
		if err != nil {
			return nil, err
		}
		models = append(models, model.Interface())
	}

	return models, nil
}

func unmarshalNode(data *Node, model reflect.Value, included *map[string]*Node) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("data is not a jsonapi representation of '%v'", model.Type())
		}
	}()

	modelValue := model.Elem()
	modelType := model.Type().Elem()

	var er error

	for i := 0; i < modelValue.NumField(); i++ {
		fieldType := modelType.Field(i)
		tag := fieldType.Tag.Get("jsonapi")
		if tag == "" {
			continue
		}

		fieldValue := modelValue.Field(i)

		args := strings.Split(tag, ",")

		if len(args) < 1 {
			er = ErrBadJSONAPIStructTag
			break
		}

		annotation := args[0]

		if (annotation == annotationClientID && len(args) != 1) ||
			(annotation != annotationClientID && len(args) < 2) {
			er = ErrBadJSONAPIStructTag
			break
		}

		if annotation == annotationPrimary {
			if data.ID == "" {
				continue
			}

			// Check the JSON API Type
			if data.Type != args[1] {
				er = fmt.Errorf(
					"Trying to Unmarshal an object of type %#v, but %#v does not match",
					data.Type,
					args[1],
				)
				break
			}

			// ID will have to be transmitted as astring per the JSON API spec
			v := reflect.ValueOf(data.ID)

			// Deal with PTRS
			var kind reflect.Kind
			if fieldValue.Kind() == reflect.Ptr {
				kind = fieldType.Type.Elem().Kind()
			} else {
				kind = fieldType.Type.Kind()
			}

			// Handle String case
			if kind == reflect.String {
				assign(fieldValue, v)
				continue
			}

			// Value was not a string... only other supported type was a numeric,
			// which would have been sent as a float value.
			floatValue, err := strconv.ParseFloat(data.ID, 64)
			if err != nil {
				// Could not convert the value in the "id" attr to a float
				er = ErrBadJSONAPIID
				break
			}

			err = unmarshalNumber(floatValue, fieldValue, fieldValue.Type())
			if err != nil {
				// We had a JSON float (numeric), but our field was not one of the
				// allowed numeric types
				er = ErrBadJSONAPIID
				break
			}
		} else if annotation == annotationClientID {
			if data.ClientID == "" {
				continue
			}

			fieldValue.Set(reflect.ValueOf(data.ClientID))
		} else if annotation == annotationAttribute {
			attributes := data.Attributes
			if attributes == nil || len(data.Attributes) == 0 {
				continue
			}

			var iso8601 bool

			if len(args) > 2 {
				for _, arg := range args[2:] {
					if arg == annotationISO8601 {
						iso8601 = true
					}
				}
			}

			val := attributes[args[1]]

			// continue if the attribute was not included in the request
			if val == nil {
				continue
			}

			v := reflect.ValueOf(val)

			err := unmarshalValue(fieldValue, v, fieldType.Type, iso8601)
			if err != nil {
				er = err
				break
			}

		} else if annotation == annotationRelation {
			isSlice := fieldValue.Type().Kind() == reflect.Slice

			if data.Relationships == nil || data.Relationships[args[1]] == nil {
				continue
			}

			if isSlice {
				// to-many relationship
				relationship := new(RelationshipManyNode)

				buf := bytes.NewBuffer(nil)

				json.NewEncoder(buf).Encode(data.Relationships[args[1]])
				json.NewDecoder(buf).Decode(relationship)

				data := relationship.Data
				models := reflect.New(fieldValue.Type()).Elem()

				for _, n := range data {
					m := reflect.New(fieldValue.Type().Elem().Elem())

					if err := unmarshalNode(
						fullNode(n, included),
						m,
						included,
					); err != nil {
						er = err
						break
					}

					models = reflect.Append(models, m)
				}

				fieldValue.Set(models)
			} else {
				// to-one relationships
				relationship := new(RelationshipOneNode)

				buf := bytes.NewBuffer(nil)

				json.NewEncoder(buf).Encode(
					data.Relationships[args[1]],
				)
				json.NewDecoder(buf).Decode(relationship)

				/*
					http://jsonapi.org/format/#document-resource-object-relationships
					http://jsonapi.org/format/#document-resource-object-linkage
					relationship can have a data node set to null (e.g. to disassociate the relationship)
					so unmarshal and set fieldValue only if data obj is not null
				*/
				if relationship.Data == nil {
					continue
				}

				m := reflect.New(fieldValue.Type().Elem())
				if err := unmarshalNode(
					fullNode(relationship.Data, included),
					m,
					included,
				); err != nil {
					er = err
					break
				}

				fieldValue.Set(m)

			}

		} else {
			er = fmt.Errorf(unsuportedStructTagMsg, annotation)
		}
	}

	return er
}

func unmarshalValue(fieldValue, v reflect.Value, fieldType reflect.Type, iso8601 bool) error {
	// Handle slices
	if fieldValue.Kind() == reflect.Slice {
		t := fieldValue.Type()
		sliceType := t.Elem()

		if sliceType.Kind() == reflect.Ptr {
			// Then dereference it
			sliceType = sliceType.Elem()
		}

		// []uint8 will be decoded as a base64-encoded string
		if sliceType.Kind() == reflect.Uint8 {
			decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(v.Interface().(string)))
			body, e := ioutil.ReadAll(decoder)
			if e != nil {
				return ErrInvalidBase64Str
			}

			if t.Elem().Kind() == reflect.Ptr {
				var uint8PtrSlice []*uint8

				for i := range body {
					uint8PtrSlice = append(uint8PtrSlice, &body[i])
				}

				fieldValue.Set(reflect.ValueOf(uint8PtrSlice))
			} else {
				fieldValue.Set(reflect.ValueOf(body))
			}

			return nil
		}

		values := reflect.MakeSlice(reflect.SliceOf(t.Elem()), v.Len(), v.Len())

		for i := 0; i < v.Len(); i++ {
			val := v.Index(i).Interface()
			switch fieldValue.Type().Elem() {

			// Try to unmarshal time types
			case reflect.TypeOf(time.Time{}):
				t := time.Time{}
				value := reflect.ValueOf(&t)
				e := unmarshalTime(reflect.ValueOf(val.(string)), value.Elem(), iso8601)
				if e != nil {
					return e
				}

				values.Index(i).Set(reflect.ValueOf(t))
				continue
			case reflect.TypeOf(new(time.Time)):
				t := new(time.Time)
				value := reflect.ValueOf(&t)
				e := unmarshalTimePtr(reflect.ValueOf(val.(string)), value.Elem(), iso8601)
				if e != nil {
					return e
				}

				values.Index(i).Set(reflect.ValueOf(t))
				continue
			}

			switch sliceType.Kind() {
			// If the slice type is a string, unmarshal it
			case reflect.String:
				values.Index(i).Set(reflect.ValueOf(val))
			// Attempt to unmarshal number types
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint,
				reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
				e := unmarshalNumber(val, values.Index(i), fieldValue.Type().Elem())
				if e != nil {
					return e
				}
			// No other slice types are currently supported
			default:
				return ErrUnsupportedSliceType
			}
		}

		fieldValue.Set(values)

		return nil
	}

	// Handle field of type time.Time
	if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
		return unmarshalTime(v, fieldValue, iso8601)
	}

	// Handle field of type *time.Time
	if fieldValue.Type() == reflect.TypeOf(new(time.Time)) {
		return unmarshalTimePtr(v, fieldValue, iso8601)
	}

	// JSON value was a float (numeric)
	if v.Kind() == reflect.Float64 {
		return unmarshalNumber(v.Interface(), fieldValue, fieldType)
	}

	// Field was a Pointer type
	if fieldValue.Kind() == reflect.Ptr {
		return unmarshalPtr(v, fieldValue)
	}

	// As a final catch-all, ensure types line up to avoid a runtime panic.
	if fieldValue.Kind() != v.Kind() {
		return ErrInvalidType
	}

	fieldValue.Set(reflect.ValueOf(v.Interface()))
	return nil
}

func unmarshalTime(v reflect.Value, fieldValue reflect.Value, iso8601 bool) error {
	if iso8601 {
		var tm string
		if v.Kind() == reflect.String {
			tm = v.Interface().(string)
		} else {
			return ErrInvalidISO8601
		}

		t, err := time.Parse(iso8601TimeFormat, tm)
		if err != nil {
			return ErrInvalidISO8601
		}

		fieldValue.Set(reflect.ValueOf(t))
		return nil
	}

	var at int64

	if v.Kind() == reflect.Float64 {
		at = int64(v.Interface().(float64))
	} else if v.Kind() == reflect.Int {
		at = v.Int()
	} else {
		return ErrInvalidTime
	}

	t := time.Unix(at, 0)

	fieldValue.Set(reflect.ValueOf(t))

	return nil
}

func unmarshalTimePtr(v, fieldValue reflect.Value, iso8601 bool) error {
	if iso8601 {
		var tm string
		if v.Kind() == reflect.String {
			tm = v.Interface().(string)
		} else {
			return ErrInvalidISO8601
		}

		v, err := time.Parse(iso8601TimeFormat, tm)
		if err != nil {
			return ErrInvalidISO8601
		}

		t := &v

		fieldValue.Set(reflect.ValueOf(t))

		return nil
	}

	var at int64

	if v.Kind() == reflect.Float64 {
		at = int64(v.Interface().(float64))
	} else if v.Kind() == reflect.Int {
		at = v.Int()
	} else {
		return ErrInvalidTime
	}

	unix := time.Unix(at, 0)
	t := &unix

	fieldValue.Set(reflect.ValueOf(t))

	return nil
}

func unmarshalNumber(v interface{}, fieldValue reflect.Value, fieldType reflect.Type) error {
	floatValue := v.(float64)

	// The field may or may not be a pointer to a numeric; the kind var
	// will not contain a pointer type
	var kind reflect.Kind
	if fieldValue.Kind() == reflect.Ptr {
		kind = fieldType.Elem().Kind()
	} else {
		kind = fieldType.Kind()
	}

	var numericValue reflect.Value

	switch kind {
	case reflect.Int:
		n := int(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Int8:
		n := int8(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Int16:
		n := int16(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Int32:
		n := int32(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Int64:
		n := int64(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Uint:
		n := uint(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Uint8:
		n := uint8(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Uint16:
		n := uint16(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Uint32:
		n := uint32(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Uint64:
		n := uint64(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Float32:
		n := float32(floatValue)
		numericValue = reflect.ValueOf(&n)
	case reflect.Float64:
		n := floatValue
		numericValue = reflect.ValueOf(&n)
	default:
		return ErrUnknownFieldNumberType
	}

	assign(fieldValue, numericValue)
	return nil
}

func unmarshalPtr(v, fieldValue reflect.Value) error {
	var concreteVal reflect.Value

	switch cVal := v.Interface().(type) {
	case string:
		concreteVal = reflect.ValueOf(&cVal)
	case bool:
		concreteVal = reflect.ValueOf(&cVal)
	case complex64:
		concreteVal = reflect.ValueOf(&cVal)
	case complex128:
		concreteVal = reflect.ValueOf(&cVal)
	case uintptr:
		concreteVal = reflect.ValueOf(&cVal)
	default:
		return ErrUnsupportedPtrType
	}

	if fieldValue.Type() != concreteVal.Type() {
		return ErrUnsupportedPtrType
	}

	fieldValue.Set(concreteVal)
	return nil
}

func fullNode(n *Node, included *map[string]*Node) *Node {
	includedKey := fmt.Sprintf("%s,%s", n.Type, n.ID)

	if included != nil && (*included)[includedKey] != nil {
		return (*included)[includedKey]
	}

	return n
}

// assign will take the value specified and assign it to the field; if
// field is expecting a ptr assign will assign a ptr.
func assign(field, value reflect.Value) {

	if field.Kind() == reflect.Ptr {
		field.Set(value)
	} else {
		field.Set(reflect.Indirect(value))
	}
}
