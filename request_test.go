package jsonapi

import "testing"

func TestUnmarshalSetsId(t *testing.T) {
	in := &JsonApiPayload{
		Data: &JsonApiNode{
			Id:   "5",
			Type: "blogs",
			Attributes: map[string]interface{}{
				"title":      "New blog",
				"created_at": 1436216820,
				"view_count": 1000,
			},
		},
	}
	out := new(Blog)

	if err := UnmarshalJsonApiPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if out.Id != 5 {
		t.Fatalf("Did not set Id on dst interface")
	}

	if out.CreatedAt.IsZero() {
		t.Fatalf("Did not parse time")
	}

	if out.ViewCount != 1000 {
		t.Fatalf("View count not properly serialized")
	}
}
