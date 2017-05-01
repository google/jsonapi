package jsonapi

import (
	"reflect"
	"strings"
)

type structExtractedField struct {
	reflect.Value
	reflect.Kind
	Annotation string
	Args       []string
	IsPtr      bool
}

func extractFields(model reflect.Value) ([]structExtractedField, error) {
	modelValue := model.Elem()
	modelType := model.Type().Elem()

	var fields [](structExtractedField)

	for i := 0; i < modelValue.NumField(); i++ {
		structField := modelValue.Type().Field(i)
		fieldValue := modelValue.Field(i)
		fieldType := modelType.Field(i)

		tag := structField.Tag.Get(annotationJSONAPI)
		if tag == "" {
			continue
		}
		if tag == annotationExtend && fieldType.Anonymous {
			extendedFields, er := extractFields(modelValue.Field(i).Addr())
			if er != nil {
				return nil, er
			}
			fields = append(fields, extendedFields...)
			continue
		}

		args := strings.Split(tag, annotationSeperator)

		if len(args) < 1 {
			return nil, ErrBadJSONAPIStructTag
		}

		annotation, args := args[0], args[1:]

		if (annotation == annotationClientID && len(args) != 0) ||
			(annotation != annotationClientID && len(args) < 1) {
			return nil, ErrBadJSONAPIStructTag
		}

		// Deal with PTRS
		kind := fieldValue.Kind()
		isPtr := fieldValue.Kind() == reflect.Ptr
		if isPtr {
			kind = fieldType.Type.Elem().Kind()
		}

		field := structExtractedField{Value: fieldValue, Kind: kind, Annotation: annotation, Args: args, IsPtr: isPtr}
		fields = append(fields, field)
	}

	return fields, nil
}
