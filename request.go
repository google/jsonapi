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

func unmarshalNode(data *Node, model reflect.Value, included *map[string]*Node) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("data is not a jsonapi representation of '%v'", model.Type())
		}
	}()

	modelValue := model.Elem()
	modelType := model.Type().Elem()

	var er error

	var i = 0
	modelType.FieldByNameFunc(func(name string) bool {
		if er != nil {
			return false
		}

		fieldType := modelType.Field(i)
		tag := fieldType.Tag.Get("jsonapi")
		if tag == "" {
			i += 1
			return false
		}

		fieldValue := modelValue.Field(i)

		i += 1

		args := strings.Split(tag, ",")

		if len(args) < 1 {
			er = BadJSONAPIStructTag{fieldType.Name}
			return false
		}

		annotation := args[0]

		if (annotation == "client-id" && len(args) != 1) || (annotation != "client-id" && len(args) != 2) {
			er = BadJSONAPIStructTag{fieldType.Name}
			return false
		}

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
		} else if annotation == "client-id" {
			if data.ClientId == "" {
				return false
			}

			fieldValue.Set(reflect.ValueOf(data.ClientId))
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

			if fieldValue.Type() == reflect.TypeOf(new(time.Time)) {
				var at int64

				if v.Kind() == reflect.Float64 {
					at = int64(v.Interface().(float64))
				} else if v.Kind() == reflect.Int {
					at = v.Int()
				} else {
					er = errors.New("Only numbers can be parsed as dates, unix timestamps")
					return false
				}

				v := time.Unix(at, 0)
				t := &v

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
						return false
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
					return false
				}

				fieldValue.Set(m)
			}

		} else {
			er = errors.New(fmt.Sprintf("Unsupported jsonapi tag annotation, %s", annotation))
		}

		return false
	})

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
