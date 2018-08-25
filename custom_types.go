package jsonapi

import "reflect"

type MarshallingFunc func(interface{}) (string, error)
type UnmarshallingFunc func(string) (interface{}, error)

// map of functions to use to convert the field value into a JSON string
var customTypeMarshallingFuncs map[reflect.Type]MarshallingFunc

// map of functions to use to convert the JSON string value into the target field
var customTypeUnmarshallingFuncs map[reflect.Type]UnmarshallingFunc

// init initializes the maps
func init() {
	customTypeMarshallingFuncs = make(map[reflect.Type]MarshallingFunc, 0)
	customTypeUnmarshallingFuncs = make(map[reflect.Type]UnmarshallingFunc, 0)
}

// IsRegisteredType checks if the given type `t` is registered as a custom type
func IsRegisteredType(t reflect.Type) bool {
	_, ok := customTypeMarshallingFuncs[t]
	return ok
}

// RegisterType registers the functions to convert the field from a custom type to a string and vice-versa
// in the JSON requests/responses.
// The `marshallingFunc` must be a function that returns a string (along with an error if something wrong happened)
// and the `unmarshallingFunc` must be a function that takes
// a string as its sole argument and return an instance of `typeName` (along with an error if something wrong happened).
// Eg:  `uuid.FromString(string) uuid.UUID {...} and `uuid.String() string {...}
func RegisterType(customType reflect.Type, marshallingFunc MarshallingFunc, unmarshallingFunc UnmarshallingFunc) {
	// register the pointer to the type
	customTypeMarshallingFuncs[customType] = marshallingFunc
	customTypeUnmarshallingFuncs[customType] = unmarshallingFunc
}

// resetCustomTypeRegistrations resets the custom type registration, which is useful during testing
func resetCustomTypeRegistrations() {
	customTypeMarshallingFuncs = make(map[reflect.Type]MarshallingFunc, 0)
	customTypeUnmarshallingFuncs = make(map[reflect.Type]UnmarshallingFunc, 0)
}
