package jsonapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

type Post struct {
	Id    int    `jsonapi:"primary,posts"`
	Title string `jsonapi:"attr,title"`
	Body  string `jsonapi:"attr,body"`
}

type Blog struct {
	Id          int    `jsonapi:"primary,blogs"`
	Title       string `jsonapi:"attr,title"`
	Posts       []Post `jsonapi:"relation,posts"`
	CurrentPost Post   `jsonapi:"relation,current_post"`
}

func TestHasPrimaryAnnotation(t *testing.T) {
	testModel := Blog{
		Id:    5,
		Title: "Title 1",
		Posts: []Post{
			Post{
				Id:    1,
				Title: "Foo",
				Body:  "Bar",
			},
			Post{
				Id:    2,
				Title: "Fuubar",
				Body:  "Bas",
			},
		},
		CurrentPost: Post{
			Id:    1,
			Title: "Foo",
			Body:  "Bar",
		},
	}

	resp, err := CreateJsonApiResponse(testModel)
	if err != nil {
		t.Fatal(err)
	}

	out := bytes.NewBuffer(nil)
	json.NewEncoder(out).Encode(resp)

	fmt.Printf("%s\n", out.Bytes())

	response := resp.Data

	if response.Type != "Blogs" {
		t.Fatalf("type should have been Blogs")
	}

	if response.Id != "5" {
		t.Fatalf("Id not transfered")
	}
}

func TestSupportsAttributes(t *testing.T) {
	testModel := Blog{
		Id:    5,
		Title: "Title 1",
	}

	resp, err := CreateJsonApiResponse(testModel)
	if err != nil {
		t.Fatal(err)
	}

	response := resp.Data

	if response.Attributes == nil || len(response.Attributes) != 1 {
		t.Fatalf("Expected 1 Attributes")
	}

	if response.Attributes["title"] != "Title 1" {
		t.Fatalf("Attributes hash not populated using tags correctly")
	}
}
