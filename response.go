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

var (
	ErrBadJSONAPIStructTag = errors.New("Bad jsonapi struct tag format")
	ErrBadJSONAPIID        = errors.New("id should be either string or int")
)

// MarshalOnePayload writes a jsonapi response with one, with related records sideloaded, into "included" array.
// This method encodes a response for a single record only. Hence, data will be a single record rather
// than an array of records.  If you want to serialize many records, see, MarshalManyPayload.
//
// See UnmarshalPayload for usage example.
//
// model interface{} should be a pointer to a struct.
func MarshalOnePayload(w io.Writer, model interface{}) error {
	payload, err := MarshalOne(model)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return err
	}

	return nil
}

// MarshalOne does the same as MarshalOnePayload except it just returns the payload
// and doesn't write out results.
// Useful is you use your JSON rendering library.
func MarshalOne(model interface{}) (*OnePayload, error) {
	included := make(map[string]*Node)

	rootNode, err := visitModelNode(model, &included, true)
	if err != nil {
		return nil, err
	}
	payload := &OnePayload{Data: rootNode}

	payload.Included = nodeMapValues(&included)

	return payload, nil
}

// MarshalManyPayload writes a jsonapi response with many records, with related records sideloaded, into "included" array.
// This method encodes a response for a slice of records, hence data will be an array of
// records rather than a single record.  To serialize a single record, see MarshalOnePayload
//
// For example you could pass it, w, your http.ResponseWriter, and, models, a slice of Blog
// struct instance pointers as interface{}'s to write to the response,
//
//	 func ListBlogs(w http.ResponseWriter, r *http.Request) {
//		 // ... fetch your blogs and filter, offset, limit, etc ...
//
//		 blogs := testBlogsForList()
//
//		 w.WriteHeader(200)
//		 w.Header().Set("Content-Type", "application/vnd.api+json")
//		 if err := jsonapi.MarshalManyPayload(w, blogs); err != nil {
//			 http.Error(w, err.Error(), 500)
//		 }
//	 }
//
//
// Visit https://github.com/shwoodard/jsonapi#list for more info.
//
// models []interface{} should be a slice of struct pointers.
func MarshalManyPayload(w io.Writer, models []interface{}) error {
	payload, err := MarshalMany(models)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return err
	}

	return nil
}

// MarshalMany does the same as MarshalManyPayload except it just returns the payload
// and doesn't write out results.
// Useful is you use your JSON rendering library.
func MarshalMany(models []interface{}) (*ManyPayload, error) {
	var data []*Node
	included := make(map[string]*Node)

	for i := 0; i < len(models); i++ {
		model := models[i]

		node, err := visitModelNode(model, &included, true)
		if err != nil {
			return nil, err
		}
		data = append(data, node)
	}

	if len(models) == 0 {
		data = make([]*Node, 0)
	}

	payload := &ManyPayload{
		Data:     data,
		Included: nodeMapValues(&included),
	}

	return payload, nil
}

// MarshalOnePayloadEmbedded - This method not meant to for use in implementation code, although feel
// free.  The purpose of this method is for use in tests.  In most cases, your
// request payloads for create will be embedded rather than sideloaded for related records.
// This method will serialize a single struct pointer into an embedded json
// response.  In other words, there will be no, "included", array in the json
// all relationships will be serailized inline in the data.
//
// However, in tests, you may want to construct payloads to post to create methods
// that are embedded to most closely resemble the payloads that will be produced by
// the client.  This is what this method is intended for.
//
// model interface{} should be a pointer to a struct.
func MarshalOnePayloadEmbedded(w io.Writer, model interface{}) error {
	rootNode, err := visitModelNode(model, nil, false)
	if err != nil {
		return err
	}

	payload := &OnePayload{Data: rootNode}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return err
	}

	return nil
}

func visitModelNode(model interface{}, included *map[string]*Node, sideload bool) (*Node, error) {
	node := new(Node)

	var er error

	modelValue := reflect.ValueOf(model).Elem()

	for i := 0; i < modelValue.NumField(); i++ {
		structField := modelValue.Type().Field(i)
		tag := structField.Tag.Get("jsonapi")
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
			id := fieldValue.Interface()

			switch nID := id.(type) {
			case string:
				node.ID = nID
			case int:
				node.ID = strconv.Itoa(nID)
			case int64:
				node.ID = strconv.FormatInt(nID, 10)
			case uint64:
				node.ID = strconv.FormatUint(nID, 10)
			default:
				er = ErrBadJSONAPIID
				break
			}

			node.Type = args[1]
		} else if annotation == "client-id" {
			clientID := fieldValue.String()
			if clientID != "" {
				node.ClientID = clientID
			}
		} else if annotation == "attr" {
			var omitEmpty bool

			if len(args) > 2 {
				omitEmpty = args[2] == "omitempty"
			}

			if node.Attributes == nil {
				node.Attributes = make(map[string]interface{})
			}

			if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
				t := fieldValue.Interface().(time.Time)

				if t.IsZero() {
					continue
				}

				node.Attributes[args[1]] = t.Unix()
			} else if fieldValue.Type() == reflect.TypeOf(new(time.Time)) {
				// A time pointer may be nil
				if fieldValue.IsNil() {
					if omitEmpty {
						continue
					}

					node.Attributes[args[1]] = nil
				} else {
					tm := fieldValue.Interface().(*time.Time)

					if tm.IsZero() && omitEmpty {
						continue
					}

					node.Attributes[args[1]] = tm.Unix()
				}
			} else {
				strAttr, ok := fieldValue.Interface().(string)

				if ok && strAttr == "" && omitEmpty {
					continue
				} else if ok {
					node.Attributes[args[1]] = strAttr
				} else {
					node.Attributes[args[1]] = fieldValue.Interface()
				}
			}
		} else if annotation == "relation" {
			isSlice := fieldValue.Type().Kind() == reflect.Slice

			if (isSlice && fieldValue.Len() < 1) || (!isSlice && fieldValue.IsNil()) {
				continue
			}

			if node.Relationships == nil {
				node.Relationships = make(map[string]interface{})
			}

			if isSlice {
				relationship, err := visitModelNodeRelationships(args[1], fieldValue, included, sideload)

				if err == nil {
					d := relationship.Data
					if sideload {
						var shallowNodes []*Node

						for _, n := range d {
							appendIncluded(included, n)
							shallowNodes = append(shallowNodes, toShallowNode(n))
						}

						node.Relationships[args[1]] = &RelationshipManyNode{Data: shallowNodes}
					} else {
						node.Relationships[args[1]] = relationship
					}
				} else {
					er = err
					break
				}
			} else {
				relationship, err := visitModelNode(fieldValue.Interface(), included, sideload)
				if err == nil {
					if sideload {
						appendIncluded(included, relationship)
						node.Relationships[args[1]] = &RelationshipOneNode{Data: toShallowNode(relationship)}
					} else {
						node.Relationships[args[1]] = &RelationshipOneNode{Data: relationship}
					}
				} else {
					er = err
					break
				}
			}

		} else {
			er = ErrBadJSONAPIStructTag
			break
		}
	}

	if er != nil {
		return nil, er
	}

	return node, nil
}

func toShallowNode(node *Node) *Node {
	return &Node{
		ID:   node.ID,
		Type: node.Type,
	}
}

func visitModelNodeRelationships(relationName string, models reflect.Value, included *map[string]*Node, sideload bool) (*RelationshipManyNode, error) {
	var nodes []*Node

	if models.Len() == 0 {
		nodes = make([]*Node, 0)
	}

	for i := 0; i < models.Len(); i++ {
		n := models.Index(i).Interface()
		node, err := visitModelNode(n, included, sideload)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	return &RelationshipManyNode{Data: nodes}, nil
}

func appendIncluded(m *map[string]*Node, nodes ...*Node) {
	included := *m

	for _, n := range nodes {
		k := fmt.Sprintf("%s,%s", n.Type, n.ID)

		if _, hasNode := included[k]; hasNode {
			continue
		}

		included[k] = n
	}
}

func nodeMapValues(m *map[string]*Node) []*Node {
	mp := *m
	nodes := make([]*Node, len(mp))

	i := 0
	for _, n := range mp {
		nodes[i] = n
		i++
	}

	return nodes
}
