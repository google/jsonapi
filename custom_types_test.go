package jsonapi

import (
	"reflect"
	"testing"
)

func TestRegisterCustomTypes(t *testing.T) {
	for _, uuidType := range []reflect.Type{reflect.TypeOf(UUID{}), reflect.TypeOf(&UUID{})} {
		// given
		resetCustomTypeRegistrations() // make sure no other registration interferes with this test
		// when
		RegisterType(uuidType,
			func(value interface{}) (string, error) {
				return "", nil
			},
			func(value string) (interface{}, error) {
				return nil, nil
			})
		// then
		if !IsRegisteredType(uuidType) {
			t.Fatalf("Expected `%v` to be registered but it was not", uuidType)
		}
	}
}
