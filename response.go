package jsonapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
)

func MarshalManyPayload(w io.Writer, models Models) error {
	d := models.GetData()
	data := make([]*Node, 0, len(d))

	incl := make([]*Node, 0)

	for _, model := range d {
		node, included, err := visitModelNode(model, true)
		if err != nil {
			return err
		}
		data = append(data, node)
		incl = append(incl, included...)
	}

	uniqueIncluded := make(map[string]*Node)

	for i, n := range incl {
		k := fmt.Sprintf("%s,%s", n.Type, n.Id)
		if uniqueIncluded[k] == nil {
			uniqueIncluded[k] = n
		} else {
			incl = deleteNode(incl, i)
		}
	}

	payload := &ManyPayload{
		Data:     data,
		Included: incl,
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return err
	}

	return nil
}

func MarshalOnePayloadEmbedded(model interface{}) (*OnePayload, error) {
	rootNode, _, err := visitModelNode(model, false)
	if err != nil {
		return nil, err
	}

	resp := &OnePayload{Data: rootNode}

	return resp, nil

}

func MarshalOnePayload(w io.Writer, model interface{}) error {
	rootNode, included, err := visitModelNode(model, true)
	if err != nil {
		return err
	}

	payload := &OnePayload{Data: rootNode}

	uniqueIncluded := make(map[string]*Node)

	for i, n := range included {
		k := fmt.Sprintf("%s,%s", n.Type, n.Id)
		if uniqueIncluded[k] == nil {
			uniqueIncluded[k] = n
		} else {
			included = deleteNode(included, i)
		}
	}

	payload.Included = included

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return err
	}

	return nil
}

func visitModelNode(model interface{}, sideload bool) (*Node, []*Node, error) {
	node := new(Node)

	var er error
	var included []*Node

	modelType := reflect.TypeOf(model).Elem()
	modelValue := reflect.ValueOf(model).Elem()

	var i = 0
	modelType.FieldByNameFunc(func(name string) bool {
		fieldValue := modelValue.Field(i)
		structField := modelType.Field(i)

		i += 1

		tag := structField.Tag.Get("jsonapi")

		if tag == "" {
			return false
		}

		args := strings.Split(tag, ",")

		if len(args) != 2 {
			er = errors.New(fmt.Sprintf("jsonapi tag, on %s, had two few arguments", structField.Name))
			return false
		}

		if len(args) >= 1 && args[0] != "" {
			annotation := args[0]

			if annotation == "primary" {
				node.Id = fmt.Sprintf("%v", fieldValue.Interface())
				node.Type = args[1]
			} else if annotation == "attr" {
				if node.Attributes == nil {
					node.Attributes = make(map[string]interface{})
				}

				if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
					isZeroMethod := fieldValue.MethodByName("IsZero")
					isZero := isZeroMethod.Call(make([]reflect.Value, 0))[0].Interface().(bool)
					if isZero {
						return false
					}

					unix := fieldValue.MethodByName("Unix")
					val := unix.Call(make([]reflect.Value, 0))[0]
					node.Attributes[args[1]] = val.Int()
				} else {
					node.Attributes[args[1]] = fieldValue.Interface()
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
					included = make([]*Node, 0)
				}

				if isSlice {
					relationship, err := visitModelNodeRelationships(args[1], fieldValue, sideload)
					d := relationship[args[1]].Data

					if err == nil {
						if sideload {
							shallowNodes := make([]*Node, 0)
							for _, node := range d {
								included = append(included, node)
								shallowNodes = append(shallowNodes, cloneAndRemoveAttributes(node))
							}

							node.Relationships[args[1]] = &RelationshipManyNode{Data: shallowNodes}
						} else {
							node.Relationships[args[1]] = &RelationshipManyNode{Data: d}
						}
					} else {
						er = err
						return false
					}
				} else {
					relationship, _, err := visitModelNode(fieldValue.Interface(), sideload)
					if err == nil {
						if sideload {
							included = append(included, relationship)
							node.Relationships[args[1]] = &RelationshipOneNode{Data: cloneAndRemoveAttributes(relationship)}
						} else {
							node.Relationships[args[1]] = &RelationshipOneNode{Data: relationship}
						}
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

func cloneAndRemoveAttributes(node *Node) *Node {
	n := *node
	n.Attributes = nil

	return &n
}

func visitModelNodeRelationships(relationName string, models reflect.Value, sideload bool) (map[string]*RelationshipManyNode, error) {
	m := make(map[string]*RelationshipManyNode)
	nodes := make([]*Node, 0)

	for i := 0; i < models.Len(); i++ {
		node, _, err := visitModelNode(models.Index(i).Interface(), sideload)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	m[relationName] = &RelationshipManyNode{Data: nodes}

	return m, nil
}

func deleteNode(a []*Node, i int) []*Node {
	if i < len(a)-1 {
		a = append(a[:i], a[i+1:]...)
	} else {
		a = a[:i]
	}

	return a
}
