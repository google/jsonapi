package jsonapi

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func UnmarshalJsonApiPayload(payload *JsonApiOnePayload, model interface{}) error {
	data := payload.Data

	modelType := reflect.TypeOf(model).Elem()
	modelValue := reflect.ValueOf(model).Elem()

	var er error

	var i = 0
	modelType.FieldByNameFunc(func(name string) bool {
		fieldType := modelType.Field(i)
		fieldValue := modelValue.Field(i)

		i += 1

		tag := fieldType.Tag.Get("jsonapi")

		args := strings.Split(tag, ",")

		if len(args) >= 1 && args[0] != "" {
			annotation := args[0]

			if annotation == "primary" {
				if len(args) >= 2 {
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
				} else {
					er = errors.New("'type' as second argument required for 'primary'")
					return false
				}
			} else if annotation == "attr" {
				attributes := data.Attributes
				if attributes == nil {
					return false
				}

				if len(args) >= 2 {
					val := attributes[args[1]]

					if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
						if reflect.TypeOf(val).Kind() != reflect.Int {
							er = errors.New("Cannot parse anything but int to Time")
							return false
						}

						t := time.Unix(reflect.ValueOf(val).Int(), 0)
						fieldValue.Set(reflect.ValueOf(t))

						return false
					}

					fieldValue.Set(reflect.ValueOf(val))
				} else {
					er = errors.New("Attribute key required as second arg")
				}
			}
			//} else if annotation == "relation" {

			//isSlice := fieldValue.Type().Kind() == reflect.Slice

			//if (isSlice && fieldValue.Len() < 1) || (!isSlice && fieldValue.IsNil()) {
			//return false
			//}

			//if node.Relationships == nil {
			//node.Relationships = make(map[string]interface{})
			//}

			//if included == nil {
			//included = make([]*JsonApiNode, 0)
			//}

			//if isSlice {
			//relationship, err := visitModelNodeRelationships(args[1], fieldValue)

			//if err == nil {
			//shallowNodes := make([]*JsonApiNode, 0)
			//for k, v := range relationship {
			//for _, node := range v {
			//included = append(included, node)

			//shallowNode := *node
			//shallowNode.Attributes = nil
			//shallowNodes = append(shallowNodes, &shallowNode)
			//}

			//node.Relationships[k] = shallowNodes
			//}
			//} else {
			//err = err
			//}
			//} else {
			//relationship, _, err := visitModelNode(fieldValue.Interface())
			//if err == nil {
			//shallowNode := *relationship
			//shallowNode.Attributes = nil

			//included = append(included, relationship)

			//node.Relationships[args[1]] = &shallowNode
			//} else {
			//err = err
			//}
			//}

			//} else {
			//err = errors.New(fmt.Sprintf("Unsupported jsonapi tag annotation, %s", annotation))
			//}
		}

		return false
	})

	if er != nil {
		return er
	}

	return nil
}
