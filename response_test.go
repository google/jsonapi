package jsonapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
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

func TestHasPrimaryAnnotation(t *testing.T) {
	testModel := &Blog{
		Id:        5,
		Title:     "Title 1",
		CreatedAt: time.Now(),
	}

	resp, err := MarshalJsonApiOnePayload(testModel)
	if err != nil {
		t.Fatal(err)
	}

	response := resp.Data

	if response.Type != "blogs" {
		t.Fatalf("type should have been blogs, got %s", response.Type)
	}

	if response.Id != "5" {
		t.Fatalf("Id not transfered")
	}
}

func TestSupportsAttributes(t *testing.T) {
	testModel := &Blog{
		Id:        5,
		Title:     "Title 1",
		CreatedAt: time.Now(),
	}

	resp, err := MarshalJsonApiOnePayload(testModel)
	if err != nil {
		t.Fatal(err)
	}

	response := resp.Data

	if response.Attributes == nil {
		t.Fatalf("Expected attributes")
	}

	if response.Attributes["title"] != "Title 1" {
		t.Fatalf("Attributes hash not populated using tags correctly")
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

	resp, err := MarshalJsonApiOnePayload(testModel)
	if err != nil {
		t.Fatal(err)
	}

	out := bytes.NewBuffer(nil)
	json.NewEncoder(out).Encode(resp)

	fmt.Printf("%s\n", out.Bytes())

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

	if reflect.ValueOf(relations["posts"]).Elem().FieldByName("Data").Len() != 2 {
		t.Fatalf("Did not materialize two posts")
	}
}

func TestNoRelations(t *testing.T) {
	testModel := &Blog{Id: 1, Title: "Title 1", CreatedAt: time.Now()}

	resp, err := MarshalJsonApiOnePayload(testModel)
	if err != nil {
		t.Fatal(err)
	}

	jsonBuffer := bytes.NewBuffer(nil)

	json.NewEncoder(jsonBuffer).Encode(resp)

	fmt.Printf("%s\n", jsonBuffer.Bytes())

	decodedResponse := new(JsonApiOnePayload)

	json.NewDecoder(jsonBuffer).Decode(decodedResponse)

	if decodedResponse.Included != nil {
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

	resp, err := MarshalJsonApiManyPayload(data)
	if err != nil {
		t.Fatal(err)
	}

	out := bytes.NewBuffer(nil)
	json.NewEncoder(out).Encode(resp)

	fmt.Printf("%s\n", out.Bytes())

	d := resp.Data

	if len(d) != 2 {
		t.Fatalf("data should have two elements")
	}
}
