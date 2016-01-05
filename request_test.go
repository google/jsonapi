package jsonapi

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

type BadModel struct {
	Id int `jsonapi:"primary"`
}

func TestMalformedTag(t *testing.T) {
	out := new(BadModel)
	err := UnmarshalPayload(samplePayload(), out)
	if err == nil || err != ErrBadJSONAPIStructTag {
		t.Fatalf("Did not error out with wrong number of arguments in tag")
	}
}

func TestUnmarshalInvalidJSON(t *testing.T) {
	in := strings.NewReader("{}")
	out := new(Blog)

	err := UnmarshalPayload(in, out)

	if err == nil {
		t.Fatalf("Did not error out the invalid JSON.")
	}
}

func TestUnmarshalSetsId(t *testing.T) {
	in := samplePayloadWithId()
	out := new(Blog)

	if err := UnmarshalPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if out.Id != 2 {
		t.Fatalf("Did not set Id on dst interface")
	}
}

func TestUnmarshalSetsAttrs(t *testing.T) {
	out, err := unmarshalSamplePayload()
	if err != nil {
		t.Fatal(err)
	}

	if out.CreatedAt.IsZero() {
		t.Fatalf("Did not parse time")
	}

	if out.ViewCount != 1000 {
		t.Fatalf("View count not properly serialized")
	}
}

func TestUnmarshalRelationships(t *testing.T) {
	out, err := unmarshalSamplePayload()
	if err != nil {
		t.Fatal(err)
	}

	if out.CurrentPost == nil {
		t.Fatalf("Current post was not materialized")
	}

	if out.CurrentPost.Title != "Bas" || out.CurrentPost.Body != "Fuubar" {
		t.Fatalf("Attributes were not set")
	}

	if len(out.Posts) != 2 {
		t.Fatalf("There should have been 2 posts")
	}
}

func TestUnmarshalNestedRelationships(t *testing.T) {
	out, err := unmarshalSamplePayload()
	if err != nil {
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

func TestUnmarshalRelationshipsSerializedEmbedded(t *testing.T) {
	out := sampleSerializedEmbeddedTestModel()

	if out.CurrentPost == nil {
		t.Fatalf("Current post was not materialized")
	}

	if out.CurrentPost.Title != "Foo" || out.CurrentPost.Body != "Bar" {
		t.Fatalf("Attributes were not set")
	}

	if len(out.Posts) != 2 {
		t.Fatalf("There should have been 2 posts")
	}

	if out.Posts[0].LatestComment.Body != "foo" {
		t.Fatalf("The comment body was not set")
	}
}

func TestUnmarshalNestedRelationshipsEmbedded(t *testing.T) {
	out := bytes.NewBuffer(nil)
	if err := MarshalOnePayloadEmbedded(out, testModel()); err != nil {
		t.Fatal(err)
	}

	model := new(Blog)

	if err := UnmarshalPayload(out, model); err != nil {
		t.Fatal(err)
	}

	if model.CurrentPost == nil {
		t.Fatalf("Current post was not materialized")
	}

	if model.CurrentPost.Comments == nil {
		t.Fatalf("Did not materialize nested records, comments")
	}

	if len(model.CurrentPost.Comments) != 2 {
		t.Fatalf("Wrong number of comments")
	}

	if model.CurrentPost.Comments[0].Body != "foo" {
		t.Fatalf("Comment body not set")
	}
}

func TestUnmarshalRelationshipsSideloaded(t *testing.T) {
	payload := samplePayloadWithSideloaded()
	out := new(Blog)

	if err := UnmarshalPayload(payload, out); err != nil {
		t.Fatal(err)
	}

	if out.CurrentPost == nil {
		t.Fatalf("Current post was not materialized")
	}

	if out.CurrentPost.Title != "Foo" || out.CurrentPost.Body != "Bar" {
		t.Fatalf("Attributes were not set")
	}

	if len(out.Posts) != 2 {
		t.Fatalf("There should have been 2 posts")
	}
}

func TestUnmarshalNestedRelationshipsSideloaded(t *testing.T) {
	payload := samplePayloadWithSideloaded()
	out := new(Blog)

	if err := UnmarshalPayload(payload, out); err != nil {
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

	if out.CurrentPost.Comments[0].Body != "foo" {
		t.Fatalf("Comment body not set")
	}
}

func TestUnmarshalNestedRelationshipsEmbedded_withClientIDs(t *testing.T) {
	model := new(Blog)

	if err := UnmarshalPayload(samplePayload(), model); err != nil {
		t.Fatal(err)
	}

	if model.Posts[0].ClientId == "" {
		t.Fatalf("ClientId not set from request on related record")
	}
}

func unmarshalSamplePayload() (*Blog, error) {
	in := samplePayload()
	out := new(Blog)

	if err := UnmarshalPayload(in, out); err != nil {
		return nil, err
	}

	return out, nil
}

func samplePayload() io.Reader {
	payload := &OnePayload{
		Data: &Node{
			Type: "blogs",
			Attributes: map[string]interface{}{
				"title":      "New blog",
				"created_at": 1436216820,
				"view_count": 1000,
			},
			Relationships: map[string]interface{}{
				"posts": &RelationshipManyNode{
					Data: []*Node{
						&Node{
							Type: "posts",
							Attributes: map[string]interface{}{
								"title": "Foo",
								"body":  "Bar",
							},
							ClientId: "1",
						},
						&Node{
							Type: "posts",
							Attributes: map[string]interface{}{
								"title": "X",
								"body":  "Y",
							},
							ClientId: "2",
						},
					},
				},
				"current_post": &RelationshipOneNode{
					Data: &Node{
						Type: "posts",
						Attributes: map[string]interface{}{
							"title": "Bas",
							"body":  "Fuubar",
						},
						ClientId: "3",
						Relationships: map[string]interface{}{
							"comments": &RelationshipManyNode{
								Data: []*Node{
									&Node{
										Type: "comments",
										Attributes: map[string]interface{}{
											"body": "Great post!",
										},
										ClientId: "4",
									},
									&Node{
										Type: "comments",
										Attributes: map[string]interface{}{
											"body": "Needs some work!",
										},
										ClientId: "5",
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

func samplePayloadWithId() io.Reader {
	payload := &OnePayload{
		Data: &Node{
			Id:   "2",
			Type: "blogs",
			Attributes: map[string]interface{}{
				"title":      "New blog",
				"view_count": 1000,
			},
		},
	}

	out := bytes.NewBuffer(nil)

	json.NewEncoder(out).Encode(payload)

	return out
}

func testModel() *Blog {
	return &Blog{
		Id:        5,
		ClientId:  "1",
		Title:     "Title 1",
		CreatedAt: time.Now(),
		Posts: []*Post{
			&Post{
				Id:    1,
				Title: "Foo",
				Body:  "Bar",
				Comments: []*Comment{
					&Comment{
						Id:   1,
						Body: "foo",
					},
					&Comment{
						Id:   2,
						Body: "bar",
					},
				},
				LatestComment: &Comment{
					Id:   1,
					Body: "foo",
				},
			},
			&Post{
				Id:    2,
				Title: "Fuubar",
				Body:  "Bas",
				Comments: []*Comment{
					&Comment{
						Id:   1,
						Body: "foo",
					},
					&Comment{
						Id:   3,
						Body: "bas",
					},
				},
				LatestComment: &Comment{
					Id:   1,
					Body: "foo",
				},
			},
		},
		CurrentPost: &Post{
			Id:    1,
			Title: "Foo",
			Body:  "Bar",
			Comments: []*Comment{
				&Comment{
					Id:   1,
					Body: "foo",
				},
				&Comment{
					Id:   2,
					Body: "bar",
				},
			},
			LatestComment: &Comment{
				Id:   1,
				Body: "foo",
			},
		},
	}
}

func samplePayloadWithSideloaded() io.Reader {
	testModel := testModel()

	out := bytes.NewBuffer(nil)
	MarshalOnePayload(out, testModel)

	return out
}

func sampleSerializedEmbeddedTestModel() *Blog {
	out := bytes.NewBuffer(nil)
	MarshalOnePayloadEmbedded(out, testModel())

	blog := new(Blog)

	UnmarshalPayload(out, blog)

	return blog
}
