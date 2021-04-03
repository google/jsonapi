package jsonapi

import (
	"database/sql"
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
		"id should be either string, int(8,16,32,64), uint(8,16,32,64) or sql.Null(Int32, Int64, Float64)")
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

	rootNode, err := visitModelNode(model, &included, true)
	if err != nil {
		return nil, err
	}
	payload := &OnePayload{Data: rootNode}

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
		node, err := visitModelNode(model, &included, true)
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
// be serialized inline in the data.
//
// However, in tests, you may want to construct payloads to post
// to create methods that are embedded to most closely resemble
// the payloads that will be produced by the client. This is what
// this method is intended for.
//
// model interface{} should be a pointer to a struct.
func MarshalOnePayloadEmbedded(w io.Writer, model interface{}) error {
	rootNode, err := visitModelNode(model, nil, false)
	if err != nil {
		return err
	}

	payload := &OnePayload{Data: rootNode}

	return json.NewEncoder(w).Encode(payload)
}

func visitModelNode(model interface{}, included *map[string]*Node,
	sideload bool) (*Node, error) {
	node := new(Node)

	value := reflect.ValueOf(model)
	if value.IsNil() {
		return nil, nil
	}

	modelValue := value.Elem()
	modelType := value.Type().Elem()

	for i := 0; i < modelValue.NumField(); i++ {
		structField := modelValue.Type().Field(i)
		tag := structField.Tag.Get(annotationJSONAPI)
		if tag == "" {
			continue
		}

		fieldValue := modelValue.Field(i)
		fieldType := modelType.Field(i)

		args := strings.Split(tag, annotationSeparator)

		if len(args) < 1 {
			return nil, ErrBadJSONAPIStructTag
		}

		annotation := args[0]

		if (annotation == annotationClientID && len(args) != 1) ||
			(annotation != annotationClientID && len(args) < 2) {
			return nil, ErrBadJSONAPIStructTag
		}

		var err error

		switch annotation {
		case annotationPrimary:
			node, err = resolveNodeID(node, fieldValue, fieldType)

			if err != nil {
				return nil, err
			}

			node.Type = args[1]
		case annotationClientID:
			clientID := fieldValue.String()
			if clientID != "" {
				node.ClientID = clientID
			}
		case annotationAttribute:
			node = resolveNodeAttribute(node, fieldValue, args)
		case annotationRelation:
			node, err = resolveNodeRelation(node, fieldValue, args, model, included, sideload)

			if err != nil {
				return nil, err
			}
		default:
			return nil, ErrBadJSONAPIStructTag
		}
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

func resolveNodeID(node *Node, fieldValue reflect.Value, structField reflect.StructField) (*Node, error) {
	v := fieldValue

	// Deal with PTRS
	var kind reflect.Kind
	if fieldValue.Kind() == reflect.Ptr {
		kind = structField.Type.Elem().Kind()
		v = reflect.Indirect(fieldValue)
	} else {
		kind = structField.Type.Kind()
	}

	// Handle allowed types
	switch kind {
	case reflect.String:
		node.ID = v.Interface().(string)
	case reflect.Int:
		node.ID = strconv.FormatInt(int64(v.Interface().(int)), 10)
	case reflect.Int8:
		node.ID = strconv.FormatInt(int64(v.Interface().(int8)), 10)
	case reflect.Int16:
		node.ID = strconv.FormatInt(int64(v.Interface().(int16)), 10)
	case reflect.Int32:
		node.ID = strconv.FormatInt(int64(v.Interface().(int32)), 10)
	case reflect.Int64:
		node.ID = strconv.FormatInt(v.Interface().(int64), 10)
	case reflect.Uint:
		node.ID = strconv.FormatUint(uint64(v.Interface().(uint)), 10)
	case reflect.Uint8:
		node.ID = strconv.FormatUint(uint64(v.Interface().(uint8)), 10)
	case reflect.Uint16:
		node.ID = strconv.FormatUint(uint64(v.Interface().(uint16)), 10)
	case reflect.Uint32:
		node.ID = strconv.FormatUint(uint64(v.Interface().(uint32)), 10)
	case reflect.Uint64:
		node.ID = strconv.FormatUint(v.Interface().(uint64), 10)
	case reflect.Struct:
		if str, ok := v.Interface().(sql.NullString); ok {
			node.ID = str.String
			break
		}

		if i32, ok := v.Interface().(sql.NullInt32); ok {
			node.ID = strconv.FormatInt(int64(i32.Int32), 10)
			break
		}

		if i64, ok := v.Interface().(sql.NullInt64); ok {
			node.ID = strconv.FormatInt(i64.Int64, 10)
			break
		}

		if f64, ok := v.Interface().(sql.NullFloat64); ok {
			node.ID = strconv.FormatFloat(f64.Float64, 'f', -1, 64)
			break
		}

		fallthrough
	default:
		// We had a JSON float (numeric), but our field was not one of the
		// allowed numeric types
		return nil, ErrBadJSONAPIID
	}

	return node, nil
}

func resolveNodeAttribute(node *Node, fieldValue reflect.Value, args []string) *Node {
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

	switch fieldValue.Type() {
	case reflect.TypeOf(time.Time{}):
		t := fieldValue.Interface().(time.Time)

		if t.IsZero() {
			return node
		}

		if iso8601 {
			node.Attributes[args[1]] = t.UTC().Format(iso8601TimeFormat)
		} else {
			node.Attributes[args[1]] = t.Unix()
		}
	case reflect.TypeOf(new(time.Time)):
		// A time pointer may be nil
		if fieldValue.IsNil() {
			if omitEmpty {
				return node
			}

			node.Attributes[args[1]] = nil
		} else {
			t := fieldValue.Interface().(*time.Time)

			if t.IsZero() && omitEmpty {
				return node
			}

			if iso8601 {
				node.Attributes[args[1]] = t.UTC().Format(iso8601TimeFormat)
			} else {
				node.Attributes[args[1]] = t.Unix()
			}
		}
	case reflect.TypeOf(sql.NullTime{}):
		nt := fieldValue.Interface().(sql.NullTime)

		// Time is NULL
		if !nt.Valid {
			if omitEmpty {
				return node
			}

			node.Attributes[args[1]] = nil
		} else {
			if nt.Time.IsZero() {
				return node
			}

			if iso8601 {
				node.Attributes[args[1]] = nt.Time.UTC().Format(iso8601TimeFormat)
			} else {
				node.Attributes[args[1]] = nt.Time.Unix()
			}
		}
	default:
		// Dealing with a fieldValue that is not a time
		emptyValue := reflect.Zero(fieldValue.Type())

		// See if we need to omit this field
		if omitEmpty && reflect.DeepEqual(fieldValue.Interface(), emptyValue.Interface()) {
			break
		}

		// Handle remaining sql.Null* types
		if boo, ok := fieldValue.Interface().(sql.NullBool); ok {
			if boo.Valid {
				node.Attributes[args[1]] = boo.Bool
			} else {
				node.Attributes[args[1]] = nil
			}
			break
		}

		if str, ok := fieldValue.Interface().(sql.NullString); ok {
			if str.Valid {
				node.Attributes[args[1]] = str.String
			} else {
				node.Attributes[args[1]] = nil
			}
			break
		}

		if f64, ok := fieldValue.Interface().(sql.NullFloat64); ok {
			if f64.Valid {
				node.Attributes[args[1]] = f64.Float64
			} else {
				node.Attributes[args[1]] = nil
			}
			break
		}

		if i32, ok := fieldValue.Interface().(sql.NullInt32); ok {
			if i32.Valid {
				node.Attributes[args[1]] = i32.Int32
			} else {
				node.Attributes[args[1]] = nil
			}
			break
		}

		if i64, ok := fieldValue.Interface().(sql.NullInt64); ok {
			if i64.Valid {
				node.Attributes[args[1]] = i64.Int64
			} else {
				node.Attributes[args[1]] = nil
			}
			break
		}

		// Handle string and remaining types
		if str, ok := fieldValue.Interface().(string); ok {
			node.Attributes[args[1]] = str
		} else {
			node.Attributes[args[1]] = fieldValue.Interface()
		}
	}

	return node
}

func resolveNodeRelation(node *Node, fieldValue reflect.Value, args []string,
	model interface{}, included *map[string]*Node, sideload bool) (*Node, error) {
	var omitEmpty bool

	// add support for 'omitempty' struct tag for marshaling as absent
	if len(args) > 2 {
		omitEmpty = args[2] == annotationOmitEmpty
	}

	isSlice := fieldValue.Type().Kind() == reflect.Slice
	if omitEmpty &&
		(isSlice && fieldValue.Len() < 1 ||
			(!isSlice && fieldValue.IsNil())) {
		return node, nil
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
			return nil, err
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

		return node, nil
	}

	// to-one relationships

	// Handle null relationship case
	if fieldValue.IsNil() {
		node.Relationships[args[1]] = &RelationshipOneNode{Data: nil}

		return node, nil
	}

	relationship, err := visitModelNode(
		fieldValue.Interface(),
		included,
		sideload,
	)
	if err != nil {
		return nil, err
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
