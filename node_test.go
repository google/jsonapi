package jsonapi

import (
	"reflect"
	"testing"
)

func TestHandleNodeErrors(t *testing.T) {
	tests := []struct {
		name         string
		hasNodeError bool
		input        *Node
		expected     *Node
	}{
		{
			name:         "has no errors",
			hasNodeError: false,
			input:        &Node{Attributes: attributes{"foo": true, "bar": false}},
			expected:     &Node{Attributes: attributes{"foo": true, "bar": false}},
		},
		{
			name:         "has an error",
			hasNodeError: true,
			input:        &Node{Attributes: attributes{"foo": true, "bar": &dominantFieldConflict{key: "bar", vals: []interface{}{true, false}}}},
			expected:     &Node{Attributes: attributes{"foo": true}},
		},
		{
			name:         "has a couple errors",
			hasNodeError: true,
			input: &Node{
				Attributes: attributes{
					"foo": true,
					"bar": &dominantFieldConflict{key: "bar", vals: []interface{}{true, false}},
					"bat": &dominantFieldConflict{key: "bat", vals: []interface{}{true, false}},
				},
			},
			expected: &Node{Attributes: attributes{"foo": true}},
		},
	}

	for _, scenario := range tests {
		t.Logf("scenario: %s\n", scenario.name)
		if scenario.hasNodeError && reflect.DeepEqual(scenario.input, scenario.expected) {
			t.Error("expected input and expected to be different")
		}
		scenario.input.handleNodeErrors()
		if !reflect.DeepEqual(scenario.input, scenario.expected) {
			t.Errorf("Got\n%#v\nExpected\n%#v\n", scenario.input, scenario.expected)
		}
	}
}

func TestSetAttributes(t *testing.T) {
	key := "foo"
	attr := attributes{}

	// set first val
	attr.set(key, false)

	// check presence of first val
	val, ok := attr[key]
	if !ok {
		t.Errorf("expected attributes to have key: %s\n", key)
	}

	// assert first val is not an error
	_, ok = val.(nodeError)
	if ok {
		t.Errorf("val stored for key (%s) should NOT be a nodeError\n", key)
	}

	// add second val; same key
	attr.set(key, true)

	// assert val converted to an error
	_, ok = attr[key].(nodeError)
	if !ok {
		t.Errorf("val stored for key (%s) should be a nodeError\n", key)
	}

	// add third val; same key
	attr.set(key, nil)

	// assert val remains an error
	_, ok = attr[key].(nodeError)
	if !ok {
		t.Errorf("val stored for key (%s) should be a nodeError\n", key)
	}
}

func TestMergeNode(t *testing.T) {
	parent := &Node{
		Type:       "Good",
		ID:         "99",
		Attributes: map[string]interface{}{"fizz": "buzz"},
	}

	child := &Node{
		Type:       "Better",
		ClientID:   "1111",
		Attributes: map[string]interface{}{"timbuk": 2},
	}

	expected := &Node{
		Type:       "Better",
		ID:         "99",
		ClientID:   "1111",
		Attributes: map[string]interface{}{"fizz": "buzz", "timbuk": 2},
	}

	parent.merge(child)

	if !reflect.DeepEqual(expected, parent) {
		t.Errorf("Got %+v Expected %+v", parent, expected)
	}
}

func TestCombinePeerNodesNoConflict(t *testing.T) {
	brother := &Node{
		Type:          "brother",
		ID:            "99",
		ClientID:      "9999",
		Attributes:    map[string]interface{}{"fizz": "buzz"},
		Relationships: map[string]interface{}{"father": "Joe"},
	}

	sister := &Node{
		Type:          "sister",
		ID:            "11",
		ClientID:      "1111",
		Attributes:    map[string]interface{}{"timbuk": 2},
		Relationships: map[string]interface{}{"mother": "Mary"},
	}

	expected := &Node{
		Type:          "sister",
		ID:            "11",
		ClientID:      "1111",
		Attributes:    map[string]interface{}{"fizz": "buzz", "timbuk": 2},
		Relationships: map[string]interface{}{"father": "Joe", "mother": "Mary"},
	}

	actual := combinePeerNodes([]*Node{brother, sister})

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got %+v Expected %+v", actual, expected)
	}
}
func TestCombinePeerNodesWithConflict(t *testing.T) {
	brother := &Node{
		Type:          "brother",
		ID:            "99",
		ClientID:      "9999",
		Attributes:    map[string]interface{}{"timbuk": 2},
		Relationships: map[string]interface{}{"father": "Joe"},
	}

	sister := &Node{
		Type:          "sister",
		ID:            "11",
		ClientID:      "1111",
		Attributes:    map[string]interface{}{"timbuk": 2},
		Relationships: map[string]interface{}{"mother": "Mary"},
	}

	expected := &Node{
		Type:          "sister",
		ID:            "11",
		ClientID:      "1111",
		Attributes:    map[string]interface{}{"timbuk": &dominantFieldConflict{key: "timbuk", vals: []interface{}{2, 2}}},
		Relationships: map[string]interface{}{"father": "Joe", "mother": "Mary"},
	}

	actual := combinePeerNodes([]*Node{brother, sister})

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got %+v Expected %+v", actual, expected)
	}
}

func TestDeepCopyNode(t *testing.T) {
	source := &Node{
		Type:          "thing",
		ID:            "1",
		ClientID:      "2",
		Attributes:    attributes{"key": "val"},
		Relationships: map[string]interface{}{"key": "val"},
		Links:         &Links{"key": "val"},
		Meta:          &Meta{"key": "val"},
	}

	badCopy := *source
	if !reflect.DeepEqual(badCopy, *source) {
		t.Errorf("Got %+v Expected %+v", badCopy, *source)
	}
	if reflect.ValueOf(badCopy.Attributes).Pointer() != reflect.ValueOf(source.Attributes).Pointer() {
		t.Error("Expected map address to be the same")
	}
	if reflect.ValueOf(badCopy.Relationships).Pointer() != reflect.ValueOf(source.Relationships).Pointer() {
		t.Error("Expected map address to be the same")
	}
	if reflect.ValueOf(badCopy.Links).Pointer() != reflect.ValueOf(source.Links).Pointer() {
		t.Error("Expected map address to be the same")
	}
	if reflect.ValueOf(badCopy.Meta).Pointer() != reflect.ValueOf(source.Meta).Pointer() {
		t.Error("Expected map address to be the same")
	}

	deepCopy := deepCopyNode(source)
	if !reflect.DeepEqual(*deepCopy, *source) {
		t.Errorf("Got %+v Expected %+v", *deepCopy, *source)
	}
	if reflect.ValueOf(deepCopy.Attributes).Pointer() == reflect.ValueOf(source.Attributes).Pointer() {
		t.Error("Expected map address to be different")
	}
	if reflect.ValueOf(deepCopy.Relationships).Pointer() == reflect.ValueOf(source.Relationships).Pointer() {
		t.Error("Expected map address to be different")
	}
	if reflect.ValueOf(deepCopy.Links).Pointer() == reflect.ValueOf(source.Links).Pointer() {
		t.Error("Expected map address to be different")
	}
	if reflect.ValueOf(deepCopy.Meta).Pointer() == reflect.ValueOf(source.Meta).Pointer() {
		t.Error("Expected map address to be different")
	}

}
