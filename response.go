package jsonapi

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

func MarshalJsonApiPayload(model interface{}) (*JsonApiPayload, error) {
	rootNode, included, err := visitModelNode(model)
	if err != nil {
		return nil, err
	}

	resp := &JsonApiPayload{Data: rootNode}

	uniqueIncluded := make(map[string]*JsonApiNode)

	for i, n := range included {
		k := fmt.Sprintf("%s,%s", n.Type, n.Id)
		if uniqueIncluded[k] == nil {
			uniqueIncluded[k] = n
		} else {
			included = append(included[:i], included[i+1:]...)
		}
	}

	resp.Included = included

	return resp, nil
}

func visitModelNode(model interface{}) (*JsonApiNode, []*JsonApiNode, error) {
	node := new(JsonApiNode)

	var er error
	var included []*JsonApiNode

	modelType := reflect.TypeOf(model).Elem()
	modelValue := reflect.ValueOf(model).Elem()

	var i = 0
	modelType.FieldByNameFunc(func(name string) bool {
		fieldValue := modelValue.Field(i)
		structField := modelType.Field(i)

		i += 1

		tag := structField.Tag.Get("jsonapi")

		args := strings.Split(tag, ",")

		if len(args) >= 1 && args[0] != "" {
			annotation := args[0]

			if annotation == "primary" {
				if len(args) >= 2 {
					node.Id = fmt.Sprintf("%v", fieldValue.Interface())
					node.Type = args[1]
				} else {
					er = errors.New("'type' as second argument required for 'primary'")
					return false
				}
			} else if annotation == "attr" {
				if node.Attributes == nil {
					node.Attributes = make(map[string]interface{})
				}

				if len(args) >= 2 {
					if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
						unix := fieldValue.MethodByName("Unix")
						val := unix.Call(make([]reflect.Value, 0))[0]
						node.Attributes[args[1]] = val.Int()
					} else {
						node.Attributes[args[1]] = fieldValue.Interface()
					}
				} else {
					er = errors.New("'type' as second argument required for 'primary'")
					return false
				}
			} else if annotation == "relation" {

				isSlice := fieldValue.Type().Kind() == reflect.Slice

				if (isSlice && fieldValue.Len() < 1) || (!isSlice && fieldValue.IsNil()) {
					return false
				}

				if node.Relationships == nil {
					node.Relationships = make(map[string]interface{})
				}

				if included == nil {
					included = make([]*JsonApiNode, 0)
				}

				if isSlice {
					relationship, err := visitModelNodeRelationships(args[1], fieldValue)
					d := relationship[args[1]].Data

					if err == nil {
						shallowNodes := make([]*JsonApiNode, 0)
						for _, node := range d {
							included = append(included, node)

							shallowNode := *node
							shallowNode.Attributes = nil
							shallowNodes = append(shallowNodes, &shallowNode)

						}

						node.Relationships[args[1]] = &JsonApiRelationshipMultipleNode{Data: shallowNodes}
					} else {
						er = err
						return false
					}
				} else {
					relationship, _, err := visitModelNode(fieldValue.Interface())
					if err == nil {
						shallowNode := *relationship
						shallowNode.Attributes = nil

						included = append(included, relationship)

						node.Relationships[args[1]] = &JsonApiRelationshipSingleNode{Data: &shallowNode}
					} else {
						er = err
						return false
					}
				}

			} else {
				er = errors.New(fmt.Sprintf("Unsupported jsonapi tag annotation, %s", annotation))
				return false
			}
		}

		return false
	})

	if er != nil {
		return nil, nil, er
	}

	return node, included, nil
}

func visitModelNodeRelationships(relationName string, models reflect.Value) (map[string]*JsonApiRelationshipMultipleNode, error) {
	m := make(map[string]*JsonApiRelationshipMultipleNode)
	nodes := make([]*JsonApiNode, 0)

	for i := 0; i < models.Len(); i++ {
		node, _, err := visitModelNode(models.Index(i).Interface())
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	m[relationName] = &JsonApiRelationshipMultipleNode{Data: nodes}

	return m, nil
}
