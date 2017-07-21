package jsonapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	// ErrUnsupportedPtrType is returned when the Struct field was a pointer but
	// the JSON value was of a different type
	ErrUnsupportedPtrType = errors.New("Pointer type in struct is not supported")
	// ErrInvalidType is returned when the given type is incompatible with the expected type.
	ErrInvalidType = errors.New("Invalid type provided") // I wish we used punctuation.
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

// unmarshalNode handles embedded struct models from top to down.
// it loops through the struct fields, handles attributes/relations at that level first
// the handling the embedded structs are done last, so that you get the expected composition behavior
// data (*Node) attributes are cleared on each success.
// relations/sideloaded models use deeply copied Nodes (since those sideloaded models can be referenced in multiple relations)
func unmarshalNode(data *Node, model reflect.Value, included *map[string]*Node) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("data is not a jsonapi representation of '%v'", model.Type())
		}
	}()

	modelValue := model.Elem()
	modelType := model.Type().Elem()

	type embedded struct {
		structField, model reflect.Value
	}
	embeddeds := []*embedded{}

	for i := 0; i < modelValue.NumField(); i++ {
		fieldType := modelType.Field(i)
		fieldValue := modelValue.Field(i)
		tag := fieldType.Tag.Get(annotationJSONAPI)

		// handle explicit ignore annotation
		if shouldIgnoreField(tag) {
			continue
		}

		// handles embedded structs
		if isEmbeddedStruct(fieldType) {
			embeddeds = append(embeddeds,
				&embedded{
					model:       reflect.ValueOf(fieldValue.Addr().Interface()),
					structField: fieldValue,
				},
			)
			continue
		}

		// handles pointers to embedded structs
		if isEmbeddedStructPtr(fieldType) {
			embeddeds = append(embeddeds,
				&embedded{
					model:       reflect.ValueOf(fieldValue.Interface()),
					structField: fieldValue,
				},
			)
			continue
		}

		// handle tagless; after handling embedded structs (which could be tagless)
		if tag == "" {
			continue
		}

		args := strings.Split(tag, annotationSeperator)
		// require atleast 1
		if len(args) < 1 {
			return ErrBadJSONAPIStructTag
		}

		// args[0] == annotation
		switch args[0] {
		case annotationClientID:
			if err := handleClientIDUnmarshal(data, args, fieldValue); err != nil {
				return err
			}
		case annotationPrimary:
			if err := handlePrimaryUnmarshal(data, args, fieldType, fieldValue); err != nil {
				return err
			}
		case annotationAttribute:
			if err := handleAttributeUnmarshal(data, args, fieldType, fieldValue); err != nil {
				return err
			}
		case annotationRelation:
			if err := handleRelationUnmarshal(data, args, fieldValue, included); err != nil {
				return err
			}
		default:
			return fmt.Errorf(unsuportedStructTagMsg, args[0])
		}
	}

	// handle embedded last
	for _, em := range embeddeds {
		// if nil, need to construct and rollback accordingly
		if em.model.IsNil() {
			copy := deepCopyNode(data)
			tmp := reflect.New(em.model.Type().Elem())
			if err := unmarshalNode(copy, tmp, included); err != nil {
				return err
			}

			// had changes; assign value to struct field, replace orig node (data) w/ mutated copy
			if !reflect.DeepEqual(copy, data) {
				assign(em.structField, tmp)
				data = copy
			}
			return nil
		}
		// handle non-nil scenarios
		return unmarshalNode(data, em.model, included)
	}

	return nil
}

func handleClientIDUnmarshal(data *Node, args []string, fieldValue reflect.Value) error {
	if len(args) != 1 {
		return ErrBadJSONAPIStructTag
	}

	if data.ClientID == "" {
		return nil
	}

	// set value and clear clientID to denote it's already been processed
	fieldValue.Set(reflect.ValueOf(data.ClientID))
	data.ClientID = ""

	return nil
}

func handlePrimaryUnmarshal(data *Node, args []string, fieldType reflect.StructField, fieldValue reflect.Value) error {
	if len(args) < 2 {
		return ErrBadJSONAPIStructTag
	}

	if data.ID == "" {
		return nil
	}

	// Check the JSON API Type
	if data.Type != args[1] {
		return fmt.Errorf(
			"Trying to Unmarshal an object of type %#v, but %#v does not match",
			data.Type,
			args[1],
		)
	}

	// Deal with PTRS
	var kind reflect.Kind
	if fieldValue.Kind() == reflect.Ptr {
		kind = fieldType.Type.Elem().Kind()
	} else {
		kind = fieldType.Type.Kind()
	}

	var idValue reflect.Value

	// Handle String case
	if kind == reflect.String {
		// ID will have to be transmitted as a string per the JSON API spec
		idValue = reflect.ValueOf(data.ID)
	} else {
		// Value was not a string... only other supported type was a numeric,
		// which would have been sent as a float value.
		floatValue, err := strconv.ParseFloat(data.ID, 64)
		if err != nil {
			// Could not convert the value in the "id" attr to a float
			return ErrBadJSONAPIID
		}

		// Convert the numeric float to one of the supported ID numeric types
		// (int[8,16,32,64] or uint[8,16,32,64])
		switch kind {
		case reflect.Int:
			n := int(floatValue)
			idValue = reflect.ValueOf(&n)
		case reflect.Int8:
			n := int8(floatValue)
			idValue = reflect.ValueOf(&n)
		case reflect.Int16:
			n := int16(floatValue)
			idValue = reflect.ValueOf(&n)
		case reflect.Int32:
			n := int32(floatValue)
			idValue = reflect.ValueOf(&n)
		case reflect.Int64:
			n := int64(floatValue)
			idValue = reflect.ValueOf(&n)
		case reflect.Uint:
			n := uint(floatValue)
			idValue = reflect.ValueOf(&n)
		case reflect.Uint8:
			n := uint8(floatValue)
			idValue = reflect.ValueOf(&n)
		case reflect.Uint16:
			n := uint16(floatValue)
			idValue = reflect.ValueOf(&n)
		case reflect.Uint32:
			n := uint32(floatValue)
			idValue = reflect.ValueOf(&n)
		case reflect.Uint64:
			n := uint64(floatValue)
			idValue = reflect.ValueOf(&n)
		default:
			// We had a JSON float (numeric), but our field was not one of the
			// allowed numeric types
			return ErrBadJSONAPIID
		}
	}

	// set value and clear ID to denote it's already been processed
	assign(fieldValue, idValue)
	data.ID = ""

	return nil
}

func handleRelationUnmarshal(data *Node, args []string, fieldValue reflect.Value, included *map[string]*Node) error {
	if len(args) < 2 {
		return ErrBadJSONAPIStructTag
	}

	if data.Relationships == nil || data.Relationships[args[1]] == nil {
		return nil
	}

	// to-one relationships
	handler := handleToOneRelationUnmarshal
	isSlice := fieldValue.Type().Kind() == reflect.Slice
	if isSlice {
		// to-many relationship
		handler = handleToManyRelationUnmarshal
	}

	v, err := handler(data.Relationships[args[1]], fieldValue.Type(), included)
	if err != nil {
		return err
	}
	// set only if there is a val since val can be null (e.g. to disassociate the relationship)
	if v != nil {
		fieldValue.Set(*v)
	}
	delete(data.Relationships, args[1])
	return nil
}

// to-one relationships
func handleToOneRelationUnmarshal(relationData interface{}, fieldType reflect.Type, included *map[string]*Node) (*reflect.Value, error) {
	relationship := new(RelationshipOneNode)

	buf := bytes.NewBuffer(nil)
	json.NewEncoder(buf).Encode(relationData)
	json.NewDecoder(buf).Decode(relationship)

	m := reflect.New(fieldType.Elem())
	/*
		http://jsonapi.org/format/#document-resource-object-relationships
		http://jsonapi.org/format/#document-resource-object-linkage
		relationship can have a data node set to null (e.g. to disassociate the relationship)
		so unmarshal and set fieldValue only if data obj is not null
	*/
	if relationship.Data == nil {
		return nil, nil
	}

	if err := unmarshalNode(
		fullNode(relationship.Data, included),
		m,
		included,
	); err != nil {
		return nil, err
	}

	return &m, nil
}

// to-many relationship
func handleToManyRelationUnmarshal(relationData interface{}, fieldType reflect.Type, included *map[string]*Node) (*reflect.Value, error) {
	relationship := new(RelationshipManyNode)

	buf := bytes.NewBuffer(nil)
	json.NewEncoder(buf).Encode(relationData)
	json.NewDecoder(buf).Decode(relationship)

	models := reflect.New(fieldType).Elem()

	rData := relationship.Data
	for _, n := range rData {
		m := reflect.New(fieldType.Elem().Elem())

		if err := unmarshalNode(
			fullNode(n, included),
			m,
			included,
		); err != nil {
			return nil, err
		}

		models = reflect.Append(models, m)
	}

	return &models, nil
}

// TODO: break this out into smaller funcs
func handleAttributeUnmarshal(data *Node, args []string, fieldType reflect.StructField, fieldValue reflect.Value) error {
	if len(args) < 2 {
		return ErrBadJSONAPIStructTag
	}
	attributes := data.Attributes
	if attributes == nil || len(data.Attributes) == 0 {
		return nil
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
		return nil
	}

	v := reflect.ValueOf(val)

	// Handle field of type time.Time
	if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
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
			delete(data.Attributes, args[1])
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
		delete(data.Attributes, args[1])
		return nil
	}

	if fieldValue.Type() == reflect.TypeOf([]string{}) {
		values := make([]string, v.Len())
		for i := 0; i < v.Len(); i++ {
			values[i] = v.Index(i).Interface().(string)
		}

		fieldValue.Set(reflect.ValueOf(values))
		delete(data.Attributes, args[1])
		return nil
	}

	if fieldValue.Type() == reflect.TypeOf(new(time.Time)) {
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
			delete(data.Attributes, args[1])
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

		v := time.Unix(at, 0)
		t := &v

		fieldValue.Set(reflect.ValueOf(t))
		delete(data.Attributes, args[1])
		return nil
	}

	// JSON value was a float (numeric)
	if v.Kind() == reflect.Float64 {
		floatValue := v.Interface().(float64)

		// The field may or may not be a pointer to a numeric; the kind var
		// will not contain a pointer type
		var kind reflect.Kind
		if fieldValue.Kind() == reflect.Ptr {
			kind = fieldType.Type.Elem().Kind()
		} else {
			kind = fieldType.Type.Kind()
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
		delete(data.Attributes, args[1])
		return nil
	}

	// Field was a Pointer type
	if fieldValue.Kind() == reflect.Ptr {
		var concreteVal reflect.Value

		switch cVal := val.(type) {
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
		delete(data.Attributes, args[1])
		return nil
	}

	// TODO: refactor the time type handlers to implement json.Unmarshaler and move this higher
	if isJSONUnmarshaler, err := handleJSONUnmarshalerAttributes(data, args, fieldValue); isJSONUnmarshaler {
		return err
	}

	// As a final catch-all, ensure types line up to avoid a runtime panic.
	// Ignore interfaces since interfaces are poly
	if fieldValue.Kind() != reflect.Interface && fieldValue.Kind() != v.Kind() {
		return ErrInvalidType
	}

	// set val and clear attribute key so its not processed again
	fieldValue.Set(reflect.ValueOf(val))
	delete(data.Attributes, args[1])
	return nil
}

func handleJSONUnmarshalerAttributes(data *Node, args []string, fieldValue reflect.Value) (bool, error) {
	val := data.Attributes[args[1]]

	// handle struct fields that implement json.Unmarshaler
	if fieldValue.Type().NumMethod() > 0 {
		jsonUnmarshaler, ok := isJSONUnmarshaler(fieldValue)
		// if field doesn't implement the json.Unmarshaler, ignore and move on
		if !ok {
			return ok, nil
		}

		b, err := json.Marshal(val)
		if err != nil {
			return ok, err
		}

		if err := jsonUnmarshaler.UnmarshalJSON(b); err != nil {
			return ok, err
		}

		// success; clear value
		delete(data.Attributes, args[1])
		return ok, nil
	}

	// field does not implement any methods, including json.Unmarshaler; continue
	return false, nil
}

func fullNode(n *Node, included *map[string]*Node) *Node {
	includedKey := fmt.Sprintf("%s,%s", n.Type, n.ID)

	if included != nil && (*included)[includedKey] != nil {
		return deepCopyNode((*included)[includedKey])
	}

	return deepCopyNode(n)
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
