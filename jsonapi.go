package jsonapi

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type JsonApiNodeWrapper struct {
	Data *JsonApiNode `json:"data"`
}

type JsonApiNode struct {
	Type          string                 `json:"type"`
	Id            string                 `json:"id"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Relationships map[string]interface{} `json:"realtionships,omitempty"`
}

type JsonApiResponse struct {
	Data     *JsonApiNode   `json:"data"`
	Included []*JsonApiNode `json:"included,omitempty"`
}

func CreateJsonApiResponse(model interface{}) (*JsonApiResponse, error) {
	rootNode, included, err := visitModelNode(model)
	if err != nil {
		return nil, err
	}

	resp := &JsonApiResponse{Data: rootNode}

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

	var err error
	included := make([]*JsonApiNode, 0)

	modelType := reflect.TypeOf(model)

	modelType.FieldByNameFunc(func(name string) bool {
		field, found := modelType.FieldByName(name)

		if found {
			fieldValue := reflect.ValueOf(model).FieldByName(name)
			tag := field.Tag.Get("jsonapi")
			args := strings.Split(tag, ",")
			if len(args) >= 1 && args[0] != "" {
				annotation := args[0]

				if annotation == "primary" {
					if len(args) >= 2 {
						node.Id = fmt.Sprintf("%v", fieldValue.Interface())
						node.Type = args[1]
					} else {
						err = errors.New("'type' as second argument required for 'primary'")
					}
				} else if annotation == "attr" {
					if node.Attributes == nil {
						node.Attributes = make(map[string]interface{})
					}

					if len(args) >= 2 {
						node.Attributes[args[1]] = fieldValue.Interface()
					} else {
						err = errors.New("'type' as second argument required for 'primary'")
					}
				} else if annotation == "relation" {
					if node.Relationships == nil {
						node.Relationships = make(map[string]interface{})
					}

					if fieldValue.Type().Kind() == reflect.Slice {
						relationship, err := visitModelNodeRelationships(args[1], fieldValue)

						if err == nil {
							shallowNodes := make([]*JsonApiNode, 0)
							for k, v := range relationship {
								for _, node := range v {
									included = append(included, node)

									shallowNode := *node
									shallowNode.Attributes = nil
									shallowNodes = append(shallowNodes, &shallowNode)
								}

								node.Relationships[k] = shallowNodes
							}
						} else {
							err = err
						}
					} else {
						relationship, _, err := visitModelNode(fieldValue.Interface())
						if err == nil {
							shallowNode := *relationship
							shallowNode.Attributes = nil

							included = append(included, relationship)

							node.Relationships[args[1]] = &shallowNode
						} else {
							err = err
						}
					}

				} else {
					err = errors.New(fmt.Sprintf("Unsupported jsonapi tag annotation, %s", annotation))
				}
			}
		}

		return false
	})

	if err != nil {
		return nil, nil, err
	}

	return node, included, nil
}

func visitModelNodeRelationships(relationName string, models reflect.Value) (map[string][]*JsonApiNode, error) {
	relationship := make(map[string][]*JsonApiNode)
	nodes := make([]*JsonApiNode, 0)

	for i := 0; i < models.Len(); i++ {
		node, _, err := visitModelNode(models.Index(i).Interface())
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	relationship[relationName] = nodes

	return relationship, nil
}

func handleField(field reflect.StructField) {

}
