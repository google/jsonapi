package jsonapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func UnmarshalJsonApiPayload(in io.Reader, model interface{}) error {
	payload := new(JsonApiOnePayload)

	if err := json.NewDecoder(in).Decode(payload); err != nil {
		return err
	}

	return unmarshalJsonApiNode(payload.Data, reflect.ValueOf(model))
}

func unmarshalJsonApiNode(data *JsonApiNode, model reflect.Value) error {
	modelValue := model.Elem()
	modelType := model.Type().Elem()

	var er error

	var i = 0
	modelType.FieldByNameFunc(func(name string) bool {
		if er != nil {
			return false
		}

		fieldValue := modelValue.Field(i)
		fieldType := modelType.Field(i)

		i += 1

		tag := fieldType.Tag.Get("jsonapi")

		args := strings.Split(tag, ",")

		if len(args) != 2 {
			er = errors.New(fmt.Sprintf("jsonapi tag, on %s, had two few arguments", fieldType.Name))
			return false
		}

		if len(args) >= 1 && args[0] != "" {
			annotation := args[0]

			if annotation == "primary" {
				if data.Id == "" {
					return false
				}

				if data.Type != args[1] {
					er = errors.New("Trying to Unmarshal a type that does not match")
					return false
				}

				if fieldValue.Kind() == reflect.String {
					fieldValue.Set(reflect.ValueOf(data.Id))
				} else if fieldValue.Kind() == reflect.Int {
					id, err := strconv.Atoi(data.Id)
					if err != nil {
						er = err
						return false
					}
					fieldValue.SetInt(int64(id))
				} else {
					er = errors.New("Unsuppored data type for primary key, not int or string")
					return false
				}
			} else if annotation == "attr" {
				attributes := data.Attributes
				if attributes == nil {
					return false
				}

				val := attributes[args[1]]

				// next if the attribute was not included in the request
				if val == nil {
					return false
				}

				v := reflect.ValueOf(val)

				if fieldValue.Type() == reflect.TypeOf(time.Time{}) {

					var at int64

					if v.Kind() == reflect.Float64 {
						at = int64(v.Interface().(float64))
					} else if v.Kind() == reflect.Int {
						at = v.Int()
					} else {
						er = errors.New("Only numbers can be parsed as dates, unix timestamps")
						return false
					}

					t := time.Unix(at, 0)

					fieldValue.Set(reflect.ValueOf(t))

					return false
				}

				if fieldValue.Kind() == reflect.Int && v.Kind() == reflect.Float64 {
					fieldValue.Set(reflect.ValueOf(int(v.Interface().(float64))))
				} else {
					fieldValue.Set(reflect.ValueOf(val))
				}
			} else if annotation == "relation" {
				isSlice := fieldValue.Type().Kind() == reflect.Slice

				if data.Relationships == nil || data.Relationships[args[1]] == nil {
					return false
				}

				relationship := reflect.ValueOf(data.Relationships[args[1]]).Interface().(map[string]interface{})

				if isSlice {
					data := relationship["data"].([]interface{})

					models := reflect.New(fieldValue.Type()).Elem()

					for _, r := range data {
						m := reflect.New(fieldValue.Type().Elem().Elem())
						h := r.(map[string]interface{})
						if err := unmarshalJsonApiNode(mapToJsonApiNode(h), m); err != nil {
							er = err
							return false
						}
						models = reflect.Append(models, m)
					}

					fieldValue.Set(models)
				} else {
					data := relationship["data"].(interface{})

					m := reflect.New(fieldValue.Type().Elem())
					h := data.(map[string]interface{})

					if err := unmarshalJsonApiNode(mapToJsonApiNode(h), m); err != nil {
						er = err
						return false
					}

					fieldValue.Set(m)
				}

			} else {
				er = errors.New(fmt.Sprintf("Unsupported jsonapi tag annotation, %s", annotation))
			}
		}

		return false
	})

	if er != nil {
		return er
	}

	return nil
}

func mapToJsonApiNode(m map[string]interface{}) *JsonApiNode {
	node := &JsonApiNode{Type: m["type"].(string)}

	if m["id"] != nil {
		node.Id = m["id"].(string)
	}

	if m["attributes"] != nil {
		node.Attributes = m["attributes"].(map[string]interface{})
	}

	if m["relationships"] != nil {
		node.Relationships = m["relationships"].(map[string]interface{})
	}

	return node
}
