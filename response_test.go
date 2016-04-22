package jsonapi

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type Blog struct {
	Id            int       `jsonapi:"primary,blogs"`
	ClientId      string    `jsonapi:"client-id"`
	Title         string    `jsonapi:"attr,title"`
	Posts         []*Post   `jsonapi:"relation,posts"`
	CurrentPost   *Post     `jsonapi:"relation,current_post"`
	CurrentPostId int       `jsonapi:"attr,current_post_id"`
	CreatedAt     time.Time `jsonapi:"attr,created_at"`
	ViewCount     int       `jsonapi:"attr,view_count"`
}

type Post struct {
	Blog
	Id            int        `jsonapi:"primary,posts"`
	BlogId        int        `jsonapi:"attr,blog_id"`
	ClientId      string     `jsonapi:"client-id"`
	Title         string     `jsonapi:"attr,title"`
	Body          string     `jsonapi:"attr,body"`
	Comments      []*Comment `jsonapi:"relation,comments"`
	LatestComment *Comment   `jsonapi:"relation,latest_comment"`
}

type Comment struct {
	Id       int    `jsonapi:"primary,comments"`
	ClientId string `jsonapi:"client-id"`
	PostId   int    `jsonapi:"attr,post_id"`
	Body     string `jsonapi:"attr,body"`
}

func TestHasPrimaryAnnotation(t *testing.T) {
	testModel := &Blog{
		Id:        5,
		Title:     "Title 1",
		CreatedAt: time.Now(),
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalOnePayload(out, testModel); err != nil {
		t.Fatal(err)
	}

	resp := new(OnePayload)

	if err := json.NewDecoder(out).Decode(resp); err != nil {
		t.Fatal(err)
	}

	data := resp.Data

	if data.Type != "blogs" {
		t.Fatalf("type should have been blogs, got %s", data.Type)
	}

	if data.Id != "5" {
		t.Fatalf("Id not transfered")
	}
}

func TestSupportsAttributes(t *testing.T) {
	testModel := &Blog{
		Id:        5,
		Title:     "Title 1",
		CreatedAt: time.Now(),
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalOnePayload(out, testModel); err != nil {
		t.Fatal(err)
	}

	resp := new(OnePayload)
	if err := json.NewDecoder(out).Decode(resp); err != nil {
		t.Fatal(err)
	}

	data := resp.Data

	if data.Attributes == nil {
		t.Fatalf("Expected attributes")
	}

	if data.Attributes["title"] != "Title 1" {
		t.Fatalf("Attributes hash not populated using tags correctly")
	}
}

func TestOmitsZeroTimes(t *testing.T) {
	testModel := &Blog{
		Id:        5,
		Title:     "Title 1",
		CreatedAt: time.Time{},
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalOnePayload(out, testModel); err != nil {
		t.Fatal(err)
	}

	resp := new(OnePayload)
	if err := json.NewDecoder(out).Decode(resp); err != nil {
		t.Fatal(err)
	}

	data := resp.Data

	if data.Attributes == nil {
		t.Fatalf("Expected attributes")
	}

	if data.Attributes["created_at"] != nil {
		t.Fatalf("Created at was serialized even though it was a zero Time")
	}
}

func TestRelations(t *testing.T) {
	testModel := testBlog()

	out := bytes.NewBuffer(nil)
	if err := MarshalOnePayload(out, testModel); err != nil {
		t.Fatal(err)
	}

	resp := new(OnePayload)
	if err := json.NewDecoder(out).Decode(resp); err != nil {
		t.Fatal(err)
	}

	relations := resp.Data.Relationships

	if relations == nil {
		t.Fatalf("Relationships were not materialized")
	}

	if relations["posts"] == nil {
		t.Fatalf("Posts relationship was not materialized")
	}

	if relations["current_post"] == nil {
		t.Fatalf("Current post relationship was not materialized")
	}

	if len(relations["posts"].(map[string]interface{})["data"].([]interface{})) != 2 {
		t.Fatalf("Did not materialize two posts")
	}
}

func TestNoRelations(t *testing.T) {
	testModel := &Blog{Id: 1, Title: "Title 1", CreatedAt: time.Now()}

	out := bytes.NewBuffer(nil)
	if err := MarshalOnePayload(out, testModel); err != nil {
		t.Fatal(err)
	}

	resp := new(OnePayload)
	if err := json.NewDecoder(out).Decode(resp); err != nil {
		t.Fatal(err)
	}

	if resp.Included != nil {
		t.Fatalf("Encoding json response did not omit included")
	}
}

func TestMarshalMany(t *testing.T) {
	data := []interface{}{
		&Blog{
			Id:        5,
			Title:     "Title 1",
			CreatedAt: time.Now(),
			Posts: []*Post{
				&Post{
					Id:    1,
					Title: "Foo",
					Body:  "Bar",
				},
				&Post{
					Id:    2,
					Title: "Fuubar",
					Body:  "Bas",
				},
			},
			CurrentPost: &Post{
				Id:    1,
				Title: "Foo",
				Body:  "Bar",
			},
		},
		&Blog{
			Id:        6,
			Title:     "Title 2",
			CreatedAt: time.Now(),
			Posts: []*Post{
				&Post{
					Id:    3,
					Title: "Foo",
					Body:  "Bar",
				},
				&Post{
					Id:    4,
					Title: "Fuubar",
					Body:  "Bas",
				},
			},
			CurrentPost: &Post{
				Id:    4,
				Title: "Foo",
				Body:  "Bar",
			},
		},
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalManyPayload(out, data); err != nil {
		t.Fatal(err)
	}

	resp := new(ManyPayload)
	if err := json.NewDecoder(out).Decode(resp); err != nil {
		t.Fatal(err)
	}

	d := resp.Data

	if len(d) != 2 {
		t.Fatalf("data should have two elements")
	}
}

func TestErrorPayload(t *testing.T) {
	errors := []Error{
		Error{Detail: "detail1"},
		Error{Detail: "detail2"},
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalErrorPayload(out, errors...); err != nil {
		t.Fatal(err)
	}

	s := out.String()
	if !strings.Contains(s, "detail1") || !strings.Contains(s, "detail2") {
		t.Fatal("Details not found in response")
	}
}

func testBlog() *Blog {
	return &Blog{
		Id:        5,
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

type BadLinksTag struct {
	Id    int            `jsonapi:"primary,badLinks"`
	Body  string         `jsonapi:"attr,body"`
	Links map[int]string `jsonapi:"links"`
}

func TestBadLinks(t *testing.T) {
	bad := BadLinksTag{
		Id:    10,
		Body:  "body",
		Links: map[int]string{1: "link_1"},
	}

	_, err := MarshalOne(&bad)
	if err != nil && err != ErrBadJSONAPILinks {
		t.Fatalf("Unexpected error type: %+v", err)
	}
}

type GoodLinksTag struct {
	Id    int               `jsonapi:"primary,badLinks"`
	Body  string            `jsonapi:"attr,body"`
	Links map[string]string `jsonapi:"links"`
}

func TestGoodLinks(t *testing.T) {
	good := GoodLinksTag{
		Id:    10,
		Body:  "body",
		Links: map[string]string{"one": "link_1"},
	}

	data, err := MarshalOne(&good)
	if err != nil {
		t.Fatalf("Unexpected error: %+v", err)
	}

	link, ok := data.Data.Links["one"]
	if !ok {
		t.Fatal("Link not found in node")
	}
	if link != "link_1" {
		t.Fatalf("Link value is not correct: %s", link)
	}
}
