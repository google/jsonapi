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
	Included []*JsonApiNode `json:"included"`
}

func CreateJsonApiResponse(model interface{}) (*JsonApiResponse, error) {
	rootNode := new(JsonApiNode)
	jsonApiResponse := &JsonApiResponse{Data: rootNode}

	primaryKeyType := reflect.TypeOf(model)

	var err error

	primaryKeyType.FieldByNameFunc(func(name string) bool {
		field, found := primaryKeyType.FieldByName(name)

		if found {
			fieldValue := reflect.ValueOf(model).FieldByName(name)
			tag := field.Tag.Get("jsonapi")
			args := strings.Split(tag, ",")
			if len(args) >= 1 && args[0] != "" {
				annotation := args[0]

				if annotation == "primary" {
					if len(args) >= 2 {
						rootNode.Id = fmt.Sprintf("%v", fieldValue.Interface())
						rootNode.Type = args[1]
					} else {
						err = errors.New("'type' as second argument required for 'primary'")
					}
				} else if annotation == "attr" {
					if rootNode.Attributes == nil {
						rootNode.Attributes = make(map[string]interface{})
					}

					if len(args) >= 2 {
						rootNode.Attributes[args[1]] = fieldValue.Interface()
					} else {
						err = errors.New("'type' as second argument required for 'primary'")
					}

				} else {
					err = errors.New("Unsupported jsonapi tag annotation")
				}
			}
		}

		return false
	})

	if err != nil {
		return nil, err
	}

	return jsonApiResponse, nil
}

func handleField(field reflect.StructField) {

}
