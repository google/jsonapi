package jsonapi

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

type Blog struct {
	ID            int       `jsonapi:"primary,blogs"`
	ClientID      string    `jsonapi:"client-id"`
	Title         string    `jsonapi:"attr,title"`
	Posts         []*Post   `jsonapi:"relation,posts"`
	CurrentPost   *Post     `jsonapi:"relation,current_post"`
	CurrentPostID int       `jsonapi:"attr,current_post_id"`
	CreatedAt     time.Time `jsonapi:"attr,created_at"`
	ViewCount     int       `jsonapi:"attr,view_count"`
}

type Post struct {
	Blog
	ID            int        `jsonapi:"primary,posts"`
	BlogID        int        `jsonapi:"attr,blog_id"`
	ClientID      string     `jsonapi:"client-id"`
	Title         string     `jsonapi:"attr,title"`
	Body          string     `jsonapi:"attr,body"`
	Comments      []*Comment `jsonapi:"relation,comments"`
	LatestComment *Comment   `jsonapi:"relation,latest_comment"`
}

type Comment struct {
	ID       int    `jsonapi:"primary,comments"`
	ClientID string `jsonapi:"client-id"`
	PostID   int    `jsonapi:"attr,post_id"`
	Body     string `jsonapi:"attr,body"`
}

func TestHasPrimaryAnnotation(t *testing.T) {
	testModel := &Blog{
		ID:        5,
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

	if data.ID != "5" {
		t.Fatalf("ID not transfered")
	}
}

func TestSupportsAttributes(t *testing.T) {
	testModel := &Blog{
		ID:        5,
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
		ID:        5,
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
	testModel := &Blog{ID: 1, Title: "Title 1", CreatedAt: time.Now()}

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
			ID:        5,
			Title:     "Title 1",
			CreatedAt: time.Now(),
			Posts: []*Post{
				&Post{
					ID:    1,
					Title: "Foo",
					Body:  "Bar",
				},
				&Post{
					ID:    2,
					Title: "Fuubar",
					Body:  "Bas",
				},
			},
			CurrentPost: &Post{
				ID:    1,
				Title: "Foo",
				Body:  "Bar",
			},
		},
		&Blog{
			ID:        6,
			Title:     "Title 2",
			CreatedAt: time.Now(),
			Posts: []*Post{
				&Post{
					ID:    3,
					Title: "Foo",
					Body:  "Bar",
				},
				&Post{
					ID:    4,
					Title: "Fuubar",
					Body:  "Bas",
				},
			},
			CurrentPost: &Post{
				ID:    4,
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

func testBlog() *Blog {
	return &Blog{
		ID:        5,
		Title:     "Title 1",
		CreatedAt: time.Now(),
		Posts: []*Post{
			&Post{
				ID:    1,
				Title: "Foo",
				Body:  "Bar",
				Comments: []*Comment{
					&Comment{
						ID:   1,
						Body: "foo",
					},
					&Comment{
						ID:   2,
						Body: "bar",
					},
				},
				LatestComment: &Comment{
					ID:   1,
					Body: "foo",
				},
			},
			&Post{
				ID:    2,
				Title: "Fuubar",
				Body:  "Bas",
				Comments: []*Comment{
					&Comment{
						ID:   1,
						Body: "foo",
					},
					&Comment{
						ID:   3,
						Body: "bas",
					},
				},
				LatestComment: &Comment{
					ID:   1,
					Body: "foo",
				},
			},
		},
		CurrentPost: &Post{
			ID:    1,
			Title: "Foo",
			Body:  "Bar",
			Comments: []*Comment{
				&Comment{
					ID:   1,
					Body: "foo",
				},
				&Comment{
					ID:   2,
					Body: "bar",
				},
			},
			LatestComment: &Comment{
				ID:   1,
				Body: "foo",
			},
		},
	}
}
