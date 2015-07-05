package jsonapi

import "testing"

type Blog struct {
	Id    int    `json:"id" jsonapi:"primary,blogs"`
	Title string `json:"title" jsonapi:"attr,title"`
}

func TestHasPrimaryAnnotation(t *testing.T) {
	testModel := Blog{
		Id:    5,
		Title: "Title 1",
	}

	resp, err := CreateJsonApiResponse(testModel)
	response := resp.Data
	if err != nil {
		t.Fatal(err)
	}

	if response.Type != "blogs" {
		t.Fatalf("type should have been blogs")
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
	response := resp.Data
	if err != nil {
		t.Fatal(err)
	}

	if response.Attributes == nil || len(response.Attributes) != 1 {
		t.Fatalf("Expected 1 Attributes")
	}

	if response.Attributes["title"] != "Title 1" {
		t.Fatalf("Attributes hash not populated using tags correctly")
	}
}
