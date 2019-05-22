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
	// ErrBadJSONAPIStructTag is returned when the Struct field's JSON API
	// annotation is invalid.
	ErrBadJSONAPIStructTag = errors.New("Bad jsonapi struct tag format")
	// ErrBadJSONAPIID is returned when the Struct JSON API annotated "id" field
	// was not a valid numeric type.
	ErrBadJSONAPIID = errors.New(
		"id should be either string, int(8,16,32,64) or uint(8,16,32,64)")
	// ErrExpectedSlice is returned when a variable or argument was expected to
	// be a slice of *Structs; MarshalMany will return this error when its
	// interface{} argument is invalid.
	ErrExpectedSlice = errors.New("models should be a slice of struct pointers")
	// ErrUnexpectedType is returned when marshalling an interface; the interface
	// had to be a pointer or a slice; otherwise this error is returned.
	ErrUnexpectedType = errors.New("models should be a struct pointer or slice of struct pointers")
)

// MarshalPayload writes a jsonapi response for one or many records. The
// related records are sideloaded into the "included" array. If this method is
// given a struct pointer as an argument it will serialize in the form
// "data": {...}. If this method is given a slice of pointers, this method will
// serialize in the form "data": [...]
//
// One Example: you could pass it, w, your http.ResponseWriter, and, models, a
// ptr to a Blog to be written to the response body:
//
//	 func ShowBlog(w http.ResponseWriter, r *http.Request) {
//		 blog := &Blog{}
//
//		 w.Header().Set("Content-Type", jsonapi.MediaType)
//		 w.WriteHeader(http.StatusOK)
//
//		 if err := jsonapi.MarshalPayload(w, blog); err != nil {
//			 http.Error(w, err.Error(), http.StatusInternalServerError)
//		 }
//	 }
//
// Many Example: you could pass it, w, your http.ResponseWriter, and, models, a
// slice of Blog struct instance pointers to be written to the response body:
//
//	 func ListBlogs(w http.ResponseWriter, r *http.Request) {
//     blogs := []*Blog{}
//
//		 w.Header().Set("Content-Type", jsonapi.MediaType)
//		 w.WriteHeader(http.StatusOK)
//
//		 if err := jsonapi.MarshalPayload(w, blogs); err != nil {
//			 http.Error(w, err.Error(), http.StatusInternalServerError)
//		 }
//	 }
//
func MarshalPayload(w io.Writer, models interface{}) error {
	payload, err := Marshal(models)
	if err != nil {
		return err
	}

	return json.NewEncoder(w).Encode(payload)
}

// Marshal does the same as MarshalPayload except it just returns the payload
// and doesn't write out results. Useful if you use your own JSON rendering
// library.
func Marshal(models interface{}) (Payloader, error) {
	switch vals := reflect.ValueOf(models); vals.Kind() {
	case reflect.Slice:
		m, err := convertToSliceInterface(&models)
		if err != nil {
			return nil, err
		}

		payload, err := marshalMany(m)
		if err != nil {
			return nil, err
		}

		if linkableModels, isLinkable := models.(Linkable); isLinkable {
			jl := linkableModels.JSONAPILinks()
			if er := jl.validate(); er != nil {
				return nil, er
			}
			payload.Links = linkableModels.JSONAPILinks()
		}

		if metableModels, ok := models.(Metable); ok {
			payload.Meta = metableModels.JSONAPIMeta()
		}

		return payload, nil
	case reflect.Ptr:
		// Check that the pointer was to a struct
		if reflect.Indirect(vals).Kind() != reflect.Struct {
			return nil, ErrUnexpectedType
		}
		return marshalOne(models)
	default:
		return nil, ErrUnexpectedType
	}
}

// MarshalPayloadWithoutIncluded writes a jsonapi response with one or many
// records, without the related records sideloaded into "included" array.
// If you want to serialize the relations into the "included" array see
// MarshalPayload.
//
// models interface{} should be either a struct pointer or a slice of struct
// pointers.
func MarshalPayloadWithoutIncluded(w io.Writer, model interface{}) error {
	payload, err := Marshal(model)
	if err != nil {
		return err
	}
	payload.clearIncluded()

	return json.NewEncoder(w).Encode(payload)
}

// marshalOne does the same as MarshalOnePayload except it just returns the
// payload and doesn't write out results. Useful is you use your JSON rendering
// library.
func marshalOne(model interface{}) (*OnePayload, error) {
	included := make(map[string]*Node)

	node := new(Node)
	node, err := visitModelNode(model, node, &included, true)
	if err != nil {
		return nil, err
	}
	payload := &OnePayload{Data: node}

	payload.Included = nodeMapValues(&included)

	return payload, nil
}

// marshalMany does the same as MarshalManyPayload except it just returns the
// payload and doesn't write out results. Useful is you use your JSON rendering
// library.
func marshalMany(models []interface{}) (*ManyPayload, error) {
	payload := &ManyPayload{
		Data: []*Node{},
	}
	included := map[string]*Node{}

	for _, model := range models {
		node := new(Node)
		node, err := visitModelNode(model, node, &included, true)
		if err != nil {
			return nil, err
		}
		payload.Data = append(payload.Data, node)
	}
	payload.Included = nodeMapValues(&included)

	return payload, nil
}

// MarshalOnePayloadEmbedded - This method not meant to for use in
// implementation code, although feel free.  The purpose of this
// method is for use in tests.  In most cases, your request
// payloads for create will be embedded rather than sideloaded for
// related records. This method will serialize a single struct
// pointer into an embedded json response. In other words, there
// will be no, "included", array in the json all relationships will
// be serailized inline in the data.
//
// However, in tests, you may want to construct payloads to post
// to create methods that are embedded to most closely resemble
// the payloads that will be produced by the client. This is what
// this method is intended for.
//
// model interface{} should be a pointer to a struct.
func MarshalOnePayloadEmbedded(w io.Writer, model interface{}) error {
	rootNode := new(Node)
	node, err := visitModelNode(model, rootNode, nil, false)
	if err != nil {
		return err
	}

	payload := &OnePayload{Data: node}

	return json.NewEncoder(w).Encode(payload)
}

func visitModelNode(model interface{}, node *Node, included *map[string]*Node, sideload bool) (*Node, error) {
	modelValue := reflect.ValueOf(model)

	kind := modelValue.Kind()
	if (kind == reflect.Interface || kind == reflect.Ptr) && modelValue.IsNil() {
		return nil, nil
	}

	switch kind {
	case reflect.Interface:
		modelValue = modelValue.Elem()
	case reflect.Ptr:
		modelValue = reflect.Indirect(modelValue)
	}

	modelType := modelValue.Type()

	var er error

	for i := 0; i < modelValue.NumField(); i++ {
		fieldType := modelType.Field(i)
		fieldValue := modelValue.Field(i)

		if fieldType.Anonymous {
			node, er = visitModelNode(fieldValue.Interface(), node, included, sideload)
			if er != nil {
				break
			}
			continue
		}

		tag := fieldType.Tag.Get(annotationJSONAPI)
		if tag == "" {
			continue
		}

		args := strings.Split(tag, annotationSeperator)
		if len(args) < 1 {
			er = ErrBadJSONAPIStructTag
			break
		}
		annotation := args[0]

		if (annotation == annotationClientID && len(args) != 1) ||
			(annotation != annotationClientID && len(args) < 2) {
			er = ErrBadJSONAPIStructTag
			break
		}

		switch annotation {
		case annotationPrimary:
			v := fieldValue

			// Deal with pointers
			var kind reflect.Kind
			if fieldValue.Kind() == reflect.Ptr {
				kind = fieldType.Type.Elem().Kind()
				v = reflect.Indirect(fieldValue)
			} else {
				kind = fieldType.Type.Kind()
			}

			vi := v.Interface()
			// Handle allowed types
			switch kind {
			case reflect.String:
				node.ID = vi.(string)
			case reflect.Int:
				node.ID = strconv.FormatInt(int64(vi.(int)), 10)
			case reflect.Int8:
				node.ID = strconv.FormatInt(int64(vi.(int8)), 10)
			case reflect.Int16:
				node.ID = strconv.FormatInt(int64(vi.(int16)), 10)
			case reflect.Int32:
				node.ID = strconv.FormatInt(int64(vi.(int32)), 10)
			case reflect.Int64:
				node.ID = strconv.FormatInt(vi.(int64), 10)
			case reflect.Uint:
				node.ID = strconv.FormatUint(uint64(vi.(uint)), 10)
			case reflect.Uint8:
				node.ID = strconv.FormatUint(uint64(vi.(uint8)), 10)
			case reflect.Uint16:
				node.ID = strconv.FormatUint(uint64(vi.(uint16)), 10)
			case reflect.Uint32:
				node.ID = strconv.FormatUint(uint64(vi.(uint32)), 10)
			case reflect.Uint64:
				node.ID = strconv.FormatUint(vi.(uint64), 10)
			default:
				// We had a JSON float (numeric), but our field was not one of the
				// allowed numeric types
				er = ErrBadJSONAPIID
			}

			if er != nil {
				break
			}

			node.Type = args[1]
		case annotationClientID:
			clientID := fieldValue.String()
			if clientID != "" {
				node.ClientID = clientID
			}
		case annotationAttribute:
			var omitEmpty, iso8601 bool

			if len(args) > 2 {
				for _, arg := range args[2:] {
					switch arg {
					case annotationOmitEmpty:
						omitEmpty = true
					case annotationISO8601:
						iso8601 = true
					}
				}
			}

			if node.Attributes == nil {
				node.Attributes = make(map[string]interface{})
			}

			if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
				t := fieldValue.Interface().(time.Time)

				if t.IsZero() {
					continue
				}

				if iso8601 {
					node.Attributes[args[1]] = t.UTC().Format(iso8601TimeFormat)
				} else {
					node.Attributes[args[1]] = t.Unix()
				}
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

					if iso8601 {
						node.Attributes[args[1]] = tm.UTC().Format(iso8601TimeFormat)
					} else {
						node.Attributes[args[1]] = tm.Unix()
					}
				}
			} else {
				// Dealing with a fieldValue that is not a time
				emptyValue := reflect.Zero(fieldValue.Type())

				// See if we need to omit this field
				if omitEmpty && reflect.DeepEqual(fieldValue.Interface(), emptyValue.Interface()) {
					continue
				}

				strAttr, ok := fieldValue.Interface().(string)
				if ok {
					node.Attributes[args[1]] = strAttr
				} else {
					node.Attributes[args[1]] = fieldValue.Interface()
				}
			}
		case annotationRelation:
			var omitEmpty bool

			//add support for 'omitempty' struct tag for marshaling as absent
			if len(args) > 2 {
				omitEmpty = args[2] == annotationOmitEmpty
			}

			isSlice := fieldValue.Type().Kind() == reflect.Slice
			if omitEmpty &&
				(isSlice && fieldValue.Len() < 1 ||
					(!isSlice && fieldValue.IsNil())) {
				continue
			}

			if node.Relationships == nil {
				node.Relationships = make(map[string]interface{})
			}

			var relLinks *Links
			if linkableModel, ok := model.(RelationshipLinkable); ok {
				relLinks = linkableModel.JSONAPIRelationshipLinks(args[1])
			}

			var relMeta *Meta
			if metableModel, ok := model.(RelationshipMetable); ok {
				relMeta = metableModel.JSONAPIRelationshipMeta(args[1])
			}

			if isSlice {
				// to-many relationship
				relationship, err := visitModelNodeRelationships(
					fieldValue,
					included,
					sideload,
				)
				if err != nil {
					er = err
					break
				}
				relationship.Links = relLinks
				relationship.Meta = relMeta

				if sideload {
					shallowNodes := []*Node{}
					for _, n := range relationship.Data {
						appendIncluded(included, n)
						shallowNodes = append(shallowNodes, toShallowNode(n))
					}

					node.Relationships[args[1]] = &RelationshipManyNode{
						Data:  shallowNodes,
						Links: relationship.Links,
						Meta:  relationship.Meta,
					}
				} else {
					node.Relationships[args[1]] = relationship
				}
			} else {
				// to-one relationships

				// Handle null relationship case
				if fieldValue.IsNil() {
					node.Relationships[args[1]] = &RelationshipOneNode{Data: nil}
					continue
				}

				relationship := new(Node)
				relationship, err := visitModelNode(fieldValue.Interface(), relationship, included, sideload)
				if err != nil {
					er = err
					break
				}

				if sideload {
					appendIncluded(included, relationship)
					node.Relationships[args[1]] = &RelationshipOneNode{
						Data:  toShallowNode(relationship),
						Links: relLinks,
						Meta:  relMeta,
					}
				} else {
					node.Relationships[args[1]] = &RelationshipOneNode{
						Data:  relationship,
						Links: relLinks,
						Meta:  relMeta,
					}
				}
			}

		default:
			er = ErrBadJSONAPIStructTag
			break
		}
	}

	if er != nil {
		return nil, er
	}

	if linkableModel, isLinkable := model.(Linkable); isLinkable {
		jl := linkableModel.JSONAPILinks()
		if er := jl.validate(); er != nil {
			return nil, er
		}
		node.Links = linkableModel.JSONAPILinks()
	}

	if metableModel, ok := model.(Metable); ok {
		node.Meta = metableModel.JSONAPIMeta()
	}

	return node, nil
}

func toShallowNode(node *Node) *Node {
	return &Node{
		ID:   node.ID,
		Type: node.Type,
	}
}

func visitModelNodeRelationships(models reflect.Value, included *map[string]*Node,
	sideload bool) (*RelationshipManyNode, error) {
	nodes := []*Node{}

	for i := 0; i < models.Len(); i++ {
		n := models.Index(i).Interface()

		node := new(Node)
		node, err := visitModelNode(n, node, included, sideload)
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

func convertToSliceInterface(i *interface{}) ([]interface{}, error) {
	vals := reflect.ValueOf(*i)
	if vals.Kind() != reflect.Slice {
		return nil, ErrExpectedSlice
	}
	var response []interface{}
	for x := 0; x < vals.Len(); x++ {
		response = append(response, vals.Index(x).Interface())
	}
	return response, nil
}
