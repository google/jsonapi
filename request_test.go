package jsonapi

import (
	"bytes"
	"encoding/json"
	"io"
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

	//o := bytes.NewBuffer(nil)
	//json.NewEncoder(o).Encode(out)

	//fmt.Printf("%s\n", o.Bytes())

	if out.CreatedAt.IsZero() {
		t.Fatalf("Did not parse time")
	}

	if out.ViewCount != 1000 {
		t.Fatalf("View count not properly serialized")
	}
}

func TestUnmarshalRelationships(t *testing.T) {
	in := samplePayload()
	out := new(Blog)

	if err := UnmarshalJsonApiPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if out.CurrentPost == nil {
		t.Fatalf("Current post was not materialized")
	}

	if out.CurrentPost.Title != "Bas" || out.CurrentPost.Body != "Fuubar" {
		t.Fatalf("Attributes where not set")
	}

	if len(out.Posts) != 2 {
		t.Fatalf("There should have been 2 posts")
	}
}

func TestUnmarshalNestedRelationships(t *testing.T) {
	in := samplePayload()
	out := new(Blog)

	if err := UnmarshalJsonApiPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if out.CurrentPost == nil {
		t.Fatalf("Current post was not materialized")
	}

	if out.CurrentPost.Comments == nil {
		t.Fatalf("Did not materialize nested records, comments")
	}

	if len(out.CurrentPost.Comments) != 2 {
		t.Fatalf("Wrong number of comments")
	}
}

func samplePayload() io.Reader {
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
						&JsonApiNode{
							Type: "posts",
							Attributes: map[string]interface{}{
								"title": "X",
								"body":  "Y",
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
						Relationships: map[string]interface{}{
							"comments": &JsonApiRelationshipManyNode{
								Data: []*JsonApiNode{
									&JsonApiNode{
										Type: "comments",
										Attributes: map[string]interface{}{
											"body": "Great post!",
										},
									},
									&JsonApiNode{
										Type: "comments",
										Attributes: map[string]interface{}{
											"body": "Needs some work!",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	out := bytes.NewBuffer(nil)

	json.NewEncoder(out).Encode(payload)

	return out
}
