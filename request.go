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

const unsuportedStructTagMsg = "Unsupported jsonapi tag annotation, %s"

var (
	ErrInvalidTime            = errors.New("Only numbers can be parsed as dates, unix timestamps")
	ErrUnknownFieldNumberType = errors.New("The struct field was not of a known number type")
)

// Convert an io into a struct instance using jsonapi tags on struct fields.
// Method supports single request payloads only, at the moment. Bulk creates and updates
// are not supported yet.
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
//   	w.WriteHeader(201)
//   	w.Header().Set("Content-Type", "application/vnd.api+json")
//
//   	if err := jsonapi.MarshalOnePayload(w, blog); err != nil {
//   		http.Error(w, err.Error(), 500)
//   	}
//   }
//
//
// Visit https://github.com/shwoodard/jsonapi#create for more info.
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
			key := fmt.Sprintf("%s,%s", included.Type, included.Id)
			includedMap[key] = included
		}

		return unmarshalNode(payload.Data, reflect.ValueOf(model), &includedMap)
	} else {
		return unmarshalNode(payload.Data, reflect.ValueOf(model), nil)
	}

}

func UnmarshalManyPayload(in io.Reader, t reflect.Type) ([]interface{}, error) {
	payload := new(ManyPayload)

	if err := json.NewDecoder(in).Decode(payload); err != nil {
		return nil, err
	}

	if payload.Included != nil {
		includedMap := make(map[string]*Node)
		for _, included := range payload.Included {
			key := fmt.Sprintf("%s,%s", included.Type, included.Id)
			includedMap[key] = included
		}

		var models []interface{}
		for _, data := range payload.Data {
			model := reflect.New(t.Elem())
			err := unmarshalNode(data, model, &includedMap)
			if err != nil {
				return nil, err
			}
			models = append(models, model.Interface())
		}

		return models, nil
	} else {
		var models []interface{}

		for _, data := range payload.Data {
			model := reflect.New(t.Elem())
			err := unmarshalNode(data, model, nil)
			if err != nil {
				return nil, err
			}
			models = append(models, model.Interface())
		}

		return models, nil
	}

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

		if (annotation == "client-id" && len(args) != 1) || (annotation != "client-id" && len(args) < 2) {
			er = ErrBadJSONAPIStructTag
			break
		}

		if annotation == "primary" {
			if data.Id == "" {
				continue
			}

			if data.Type != args[1] {
				er = fmt.Errorf("Trying to Unmarshal an object of type %#v, but %#v does not match", data.Type, args[1])
				break
			}

			if fieldValue.Kind() == reflect.String {
				fieldValue.Set(reflect.ValueOf(data.Id))
			} else if fieldValue.Kind() == reflect.Int {
				id, err := strconv.Atoi(data.Id)
				if err != nil {
					er = err
					break
				}
				fieldValue.SetInt(int64(id))
			} else {
				er = ErrBadJSONAPIID
				break
			}
		} else if annotation == "client-id" {
			if data.ClientId == "" {
				continue
			}

			fieldValue.Set(reflect.ValueOf(data.ClientId))
		} else if annotation == "attr" {
			attributes := data.Attributes
			if attributes == nil || len(data.Attributes) == 0 {
				continue
			}

			val := attributes[args[1]]

			// continue if the attribute was not included in the request
			if val == nil {
				continue
			}

			v := reflect.ValueOf(val)

			if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
				var at int64

				if v.Kind() == reflect.Float64 {
					at = int64(v.Interface().(float64))
				} else if v.Kind() == reflect.Int {
					at = v.Int()
				} else {
					er = ErrInvalidTime
					break
				}

				t := time.Unix(at, 0)

				fieldValue.Set(reflect.ValueOf(t))

				continue
			}

			if fieldValue.Type() == reflect.TypeOf([]string(nil)) {
				values := make([]string, v.Len())
				for i := 0; i < v.Len(); i++ {
					values[i] = v.Index(i).Interface().(string)
				}

				fieldValue.Set(reflect.ValueOf(values))

				continue
			}

			if fieldValue.Type() == reflect.TypeOf(new(time.Time)) {
				var at int64

				if v.Kind() == reflect.Float64 {
					at = int64(v.Interface().(float64))
				} else if v.Kind() == reflect.Int {
					at = v.Int()
				} else {
					er = ErrInvalidTime
					break
				}

				v := time.Unix(at, 0)
				t := &v

				fieldValue.Set(reflect.ValueOf(t))

				continue
			}

			if v.Kind() == reflect.Float64 {
				// Handle JSON numeric case
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
					n := float64(floatValue)
					numericValue = reflect.ValueOf(&n)
				default:
					er = ErrUnknownFieldNumberType
					break
				}

				if fieldValue.Kind() == reflect.Ptr {
					fieldValue.Set(numericValue)
				} else {
					fieldValue.Set(reflect.Indirect(numericValue))
				}

				continue
			}
			fieldValue.Set(reflect.ValueOf(val))
		} else if annotation == "relation" {
			isSlice := fieldValue.Type().Kind() == reflect.Slice

			if data.Relationships == nil || data.Relationships[args[1]] == nil {
				continue
			}

			if isSlice {
				relationship := new(RelationshipManyNode)

				buf := bytes.NewBuffer(nil)

				json.NewEncoder(buf).Encode(data.Relationships[args[1]])
				json.NewDecoder(buf).Decode(relationship)

				data := relationship.Data
				models := reflect.New(fieldValue.Type()).Elem()

				for _, n := range data {
					m := reflect.New(fieldValue.Type().Elem().Elem())

					if err := unmarshalNode(fullNode(n, included), m, included); err != nil {
						er = err
						break
					}

					models = reflect.Append(models, m)
				}

				fieldValue.Set(models)
			} else {
				relationship := new(RelationshipOneNode)

				buf := bytes.NewBuffer(nil)

				json.NewEncoder(buf).Encode(data.Relationships[args[1]])
				json.NewDecoder(buf).Decode(relationship)

				m := reflect.New(fieldValue.Type().Elem())

				if err := unmarshalNode(fullNode(relationship.Data, included), m, included); err != nil {
					er = err
					break
				}

				fieldValue.Set(m)
			}

		} else {
			er = fmt.Errorf(unsuportedStructTagMsg, annotation)
		}
	}

	if er != nil {
		return er
	}

	return nil
}

func fullNode(n *Node, included *map[string]*Node) *Node {
	includedKey := fmt.Sprintf("%s,%s", n.Type, n.Id)

	if included != nil && (*included)[includedKey] != nil {
		return (*included)[includedKey]
	}

	return n
}
