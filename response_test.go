package jsonapi

import (
	"bytes"
	"encoding/json"
	"regexp"
	"testing"
	"time"
)

type Comment struct {
	Id   int    `jsonapi:"primary,comments"`
	Body string `jsonapi:"attr,body"`
}

type Post struct {
	Id       int        `jsonapi:"primary,posts"`
	Title    string     `jsonapi:"attr,title"`
	Body     string     `jsonapi:"attr,body"`
	Comments []*Comment `jsonapi:"relation,comments"`
}

type Blog struct {
	Id          int       `jsonapi:"primary,blogs"`
	Title       string    `jsonapi:"attr,title"`
	Posts       []*Post   `jsonapi:"relation,posts"`
	CurrentPost *Post     `jsonapi:"relation,current_post"`
	CreatedAt   time.Time `jsonapi:"attr,created_at"`
	ViewCount   int       `jsonapi:"attr,view_count"`
}

type Blogs []*Blog

func (b Blogs) GetData() []interface{} {
	d := make([]interface{}, len(b))
	for i, blog := range b {
		d[i] = blog
	}
	return d
}

func TestMalformedTagResposne(t *testing.T) {
	testModel := &BadModel{}
	out := bytes.NewBuffer(nil)
	err := MarshalJsonApiOnePayload(out, testModel)

	if err == nil {
		t.Fatalf("Did not error out with wrong number of arguments in tag")
	}

	r := regexp.MustCompile(`two few arguments`)

	if !r.Match([]byte(err.Error())) {
		t.Fatalf("The err was not due two two few arguments in a tag")
	}
}

func TestHasPrimaryAnnotation(t *testing.T) {
	testModel := &Blog{
		Id:        5,
		Title:     "Title 1",
		CreatedAt: time.Now(),
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalJsonApiOnePayload(out, testModel); err != nil {
		t.Fatal(err)
	}

	resp := new(JsonApiOnePayload)

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
	if err := MarshalJsonApiOnePayload(out, testModel); err != nil {
		t.Fatal(err)
	}

	resp := new(JsonApiOnePayload)
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
	if err := MarshalJsonApiOnePayload(out, testModel); err != nil {
		t.Fatal(err)
	}

	resp := new(JsonApiOnePayload)
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
	testModel := &Blog{
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
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalJsonApiOnePayload(out, testModel); err != nil {
		t.Fatal(err)
	}

	resp := new(JsonApiOnePayload)
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
	if err := MarshalJsonApiOnePayload(out, testModel); err != nil {
		t.Fatal(err)
	}

	resp := new(JsonApiOnePayload)
	if err := json.NewDecoder(out).Decode(resp); err != nil {
		t.Fatal(err)
	}

	if resp.Included != nil {
		t.Fatalf("Encoding json response did not omit included")
	}
}

func TestMarshalMany(t *testing.T) {
	data := Blogs{
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
	if err := MarshalJsonApiManyPayload(out, data); err != nil {
		t.Fatal(err)
	}

	resp := new(JsonApiManyPayload)
	if err := json.NewDecoder(out).Decode(resp); err != nil {
		t.Fatal(err)
	}

	d := resp.Data

	if len(d) != 2 {
		t.Fatalf("data should have two elements")
	}
}
