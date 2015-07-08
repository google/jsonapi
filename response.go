package jsonapi

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

func MarshalJsonApiManyPayload(models Models) (*JsonApiManyPayload, error) {
	d := models.GetData()
	data := make([]*JsonApiNode, 0, len(d))

	incl := make([]*JsonApiNode, 0)

	for _, model := range d {
		node, included, err := visitModelNode(model)
		if err != nil {
			return nil, err
		}
		data = append(data, node)
		incl = append(incl, included...)
	}

	uniqueIncluded := make(map[string]*JsonApiNode)

	for i, n := range incl {
		k := fmt.Sprintf("%s,%s", n.Type, n.Id)
		if uniqueIncluded[k] == nil {
			uniqueIncluded[k] = n
		} else {
			incl = deleteNode(incl, i)
		}
	}

	return &JsonApiManyPayload{
		Data:     data,
		Included: incl,
	}, nil
}

func MarshalJsonApiOnePayload(model interface{}) (*JsonApiOnePayload, error) {
	rootNode, included, err := visitModelNode(model)
	if err != nil {
		return nil, err
	}

	resp := &JsonApiOnePayload{Data: rootNode}

	uniqueIncluded := make(map[string]*JsonApiNode)

	for i, n := range included {
		k := fmt.Sprintf("%s,%s", n.Type, n.Id)
		if uniqueIncluded[k] == nil {
			uniqueIncluded[k] = n
		} else {
			included = deleteNode(included, i)
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

		if len(args) != 2 {
			er = errors.New(fmt.Sprintf("jsonapi tag, on %s, had two few arguments", structField.Name))
			return false
		}

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

						node.Relationships[args[1]] = &JsonApiRelationshipManyNode{Data: shallowNodes}
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

						node.Relationships[args[1]] = &JsonApiRelationshipOneNode{Data: &shallowNode}
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

func visitModelNodeRelationships(relationName string, models reflect.Value) (map[string]*JsonApiRelationshipManyNode, error) {
	m := make(map[string]*JsonApiRelationshipManyNode)
	nodes := make([]*JsonApiNode, 0)

	for i := 0; i < models.Len(); i++ {
		node, _, err := visitModelNode(models.Index(i).Interface())
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	m[relationName] = &JsonApiRelationshipManyNode{Data: nodes}

	return m, nil
}

func deleteNode(a []*JsonApiNode, i int) []*JsonApiNode {
	if i < len(a)-1 {
		a = append(a[:i], a[i+1:]...)
	} else {
		a = a[:i]
	}

	return a
}
