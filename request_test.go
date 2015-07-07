package jsonapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

//func TestUnmarshalSetsId(t *testing.T) {
//in := samplePayload()
//out := new(Blog)

//if err := UnmarshalJsonApiPayload(in, out); err != nil {
//t.Fatal(err)
//}

//if out.Id != 0 {
//t.Fatalf("Did not set Id on dst interface")
//}
//}

func TestUnmarshalSetsAttrs(t *testing.T) {
	in := samplePayload()
	out := new(Blog)

	if err := UnmarshalJsonApiPayload(in, out); err != nil {
		t.Fatal(err)
	}

	o := bytes.NewBuffer(nil)
	json.NewEncoder(o).Encode(out)

	fmt.Printf("%s\n", o.Bytes())

	if out.CreatedAt.IsZero() {
		t.Fatalf("Did not parse time")
	}

	if out.ViewCount != 1000 {
		t.Fatalf("View count not properly serialized")
	}
}

func samplePayload() *JsonApiOnePayload {
	payload := &JsonApiOnePayload{
		Data: &JsonApiNode{
			Type: "blogs",
			Attributes: map[string]interface{}{
				"title":      "New blog",
				"created_at": 1436216820,
				"view_count": 1000,
			},
			Relationships: map[string]interface{}{
				"posts": &JsonApiRelationshipManyNode{
					Data: []*JsonApiNode{
						&JsonApiNode{
							Type: "posts",
							Attributes: map[string]interface{}{
								"title": "Foo",
								"body":  "Bar",
							},
						},
					},
				},
				"current_post": &JsonApiRelationshipOneNode{
					Data: &JsonApiNode{
						Type: "posts",
						Attributes: map[string]interface{}{
							"title": "Bas",
							"body":  "Fuubar",
						},
					},
				},
			},
		},
	}

	out := bytes.NewBuffer(nil)

	json.NewEncoder(out).Encode(payload)

	p := new(JsonApiOnePayload)

	json.NewDecoder(out).Decode(p)

	return p
}
