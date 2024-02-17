package jsonapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestMarshalPayload(t *testing.T) {
	book := &Book{ID: 1}
	books := []*Book{book, {ID: 2}}
	var jsonData map[string]interface{}

	// One
	out1 := bytes.NewBuffer(nil)
	MarshalPayload(out1, book)

	if err := json.Unmarshal(out1.Bytes(), &jsonData); err != nil {
		t.Fatal(err)
	}
	if _, ok := jsonData["data"].(map[string]interface{}); !ok {
		t.Fatalf("data key did not contain an Hash/Dict/Map")
	}

	// Many
	out2 := bytes.NewBuffer(nil)
	MarshalPayload(out2, books)

	if err := json.Unmarshal(out2.Bytes(), &jsonData); err != nil {
		t.Fatal(err)
	}
	if _, ok := jsonData["data"].([]interface{}); !ok {
		t.Fatalf("data key did not contain an Array")
	}
}

func TestMarshalPayloadWithNulls(t *testing.T) {

	books := []*Book{nil, {ID: 101}, nil}
	var jsonData map[string]interface{}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, books); err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(out.Bytes(), &jsonData); err != nil {
		t.Fatal(err)
	}
	raw, ok := jsonData["data"]
	if !ok {
		t.Fatalf("data key does not exist")
	}
	arr, ok := raw.([]interface{})
	if !ok {
		t.Fatalf("data is not an Array")
	}
	for i := 0; i < len(arr); i++ {
		if books[i] == nil && arr[i] != nil ||
			books[i] != nil && arr[i] == nil {
			t.Fatalf("restored data is not equal to source")
		}
	}
}

func TestMarshal_attrStringSlice(t *testing.T) {
	tags := []string{"fiction", "sale"}
	b := &Book{ID: 1, Tags: tags}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, b); err != nil {
		t.Fatal(err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &jsonData); err != nil {
		t.Fatal(err)
	}

	jsonTags := jsonData["data"].(map[string]interface{})["attributes"].(map[string]interface{})["tags"].([]interface{})
	if e, a := len(tags), len(jsonTags); e != a {
		t.Fatalf("Was expecting tags of length %d got %d", e, a)
	}

	// Convert from []interface{} to []string
	jsonTagsStrings := []string{}
	for _, tag := range jsonTags {
		jsonTagsStrings = append(jsonTagsStrings, tag.(string))
	}

	// Sort both
	sort.Strings(jsonTagsStrings)
	sort.Strings(tags)

	for i, tag := range tags {
		if e, a := tag, jsonTagsStrings[i]; e != a {
			t.Fatalf("At index %d, was expecting %s got %s", i, e, a)
		}
	}
}

func TestWithoutOmitsEmptyAnnotationOnRelation(t *testing.T) {
	blog := &Blog{}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, blog); err != nil {
		t.Fatal(err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &jsonData); err != nil {
		t.Fatal(err)
	}
	relationships := jsonData["data"].(map[string]interface{})["relationships"].(map[string]interface{})

	// Verifiy the "posts" relation was an empty array
	posts, ok := relationships["posts"]
	if !ok {
		t.Fatal("Was expecting the data.relationships.posts key/value to have been present")
	}
	postsMap, ok := posts.(map[string]interface{})
	if !ok {
		t.Fatal("data.relationships.posts was not a map")
	}
	postsData, ok := postsMap["data"]
	if !ok {
		t.Fatal("Was expecting the data.relationships.posts.data key/value to have been present")
	}
	postsDataSlice, ok := postsData.([]interface{})
	if !ok {
		t.Fatal("data.relationships.posts.data was not a slice []")
	}
	if len(postsDataSlice) != 0 {
		t.Fatal("Was expecting the data.relationships.posts.data value to have been an empty array []")
	}

	// Verifiy the "current_post" was a null
	currentPost, postExists := relationships["current_post"]
	if !postExists {
		t.Fatal("Was expecting the data.relationships.current_post key/value to have NOT been omitted")
	}
	currentPostMap, ok := currentPost.(map[string]interface{})
	if !ok {
		t.Fatal("data.relationships.current_post was not a map")
	}
	currentPostData, ok := currentPostMap["data"]
	if !ok {
		t.Fatal("Was expecting the data.relationships.current_post.data key/value to have been present")
	}
	if currentPostData != nil {
		t.Fatal("Was expecting the data.relationships.current_post.data value to have been nil/null")
	}
}

func TestWithOmitsEmptyAnnotationOnRelation(t *testing.T) {
	type BlogOptionalPosts struct {
		ID          int     `jsonapi:"primary,blogs"`
		Title       string  `jsonapi:"attr,title"`
		Posts       []*Post `jsonapi:"relation,posts,omitempty"`
		CurrentPost *Post   `jsonapi:"relation,current_post,omitempty"`
	}

	blog := &BlogOptionalPosts{ID: 999}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, blog); err != nil {
		t.Fatal(err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &jsonData); err != nil {
		t.Fatal(err)
	}
	payload := jsonData["data"].(map[string]interface{})

	// Verify relationship was NOT set
	if val, exists := payload["relationships"]; exists {
		t.Fatalf("Was expecting the data.relationships key/value to have been empty - it was not and had a value of %v", val)
	}
}

func TestWithOmitsEmptyAnnotationOnRelation_MixedData(t *testing.T) {
	type BlogOptionalPosts struct {
		ID          int     `jsonapi:"primary,blogs"`
		Title       string  `jsonapi:"attr,title"`
		Posts       []*Post `jsonapi:"relation,posts,omitempty"`
		CurrentPost *Post   `jsonapi:"relation,current_post,omitempty"`
	}

	blog := &BlogOptionalPosts{
		ID: 999,
		CurrentPost: &Post{
			ID: 123,
		},
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, blog); err != nil {
		t.Fatal(err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &jsonData); err != nil {
		t.Fatal(err)
	}
	payload := jsonData["data"].(map[string]interface{})

	// Verify relationship was set
	if _, exists := payload["relationships"]; !exists {
		t.Fatal("Was expecting the data.relationships key/value to have NOT been empty")
	}

	relationships := payload["relationships"].(map[string]interface{})

	// Verify the relationship was not omitted, and is not null
	if val, exists := relationships["current_post"]; !exists {
		t.Fatal("Was expecting the data.relationships.current_post key/value to have NOT been omitted")
	} else if val.(map[string]interface{})["data"] == nil {
		t.Fatal("Was expecting the data.relationships.current_post value to have NOT been nil/null")
	}
}

func TestWithOmitsEmptyAnnotationOnAttribute(t *testing.T) {
	type Phone struct {
		Number string `json:"number"`
	}

	type Address struct {
		City   string `json:"city"`
		Street string `json:"street"`
	}

	type Tags map[string]int

	type Author struct {
		ID      int      `jsonapi:"primary,authors"`
		Name    string   `jsonapi:"attr,title"`
		Phones  []*Phone `jsonapi:"attr,phones,omitempty"`
		Address *Address `jsonapi:"attr,address,omitempty"`
		Tags    Tags     `jsonapi:"attr,tags,omitempty"`
	}

	author := &Author{
		ID:      999,
		Name:    "Igor",
		Phones:  nil,                        // should be omitted
		Address: nil,                        // should be omitted
		Tags:    Tags{"dogs": 1, "cats": 2}, // should not be omitted
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, author); err != nil {
		t.Fatal(err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &jsonData); err != nil {
		t.Fatal(err)
	}

	// Verify that there is no field "phones" in attributes
	payload := jsonData["data"].(map[string]interface{})
	attributes := payload["attributes"].(map[string]interface{})
	if _, ok := attributes["title"]; !ok {
		t.Fatal("Was expecting the data.attributes.title to have NOT been omitted")
	}
	if _, ok := attributes["phones"]; ok {
		t.Fatal("Was expecting the data.attributes.phones to have been omitted")
	}
	if _, ok := attributes["address"]; ok {
		t.Fatal("Was expecting the data.attributes.phones to have been omitted")
	}
	if _, ok := attributes["tags"]; !ok {
		t.Fatal("Was expecting the data.attributes.tags to have NOT been omitted")
	}
}

func TestMarshalIDPtr(t *testing.T) {
	id, make, model := "123e4567-e89b-12d3-a456-426655440000", "Ford", "Mustang"
	car := &Car{
		ID:    &id,
		Make:  &make,
		Model: &model,
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, car); err != nil {
		t.Fatal(err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &jsonData); err != nil {
		t.Fatal(err)
	}
	data := jsonData["data"].(map[string]interface{})
	// attributes := data["attributes"].(map[string]interface{})

	// Verify that the ID was sent
	val, exists := data["id"]
	if !exists {
		t.Fatal("Was expecting the data.id member to exist")
	}
	if val != id {
		t.Fatalf("Was expecting the data.id member to be `%s`, got `%s`", id, val)
	}
}

func TestMarshalOnePayload_omitIDString(t *testing.T) {
	type Foo struct {
		ID    string `jsonapi:"primary,foo"`
		Title string `jsonapi:"attr,title"`
	}

	foo := &Foo{Title: "Foo"}
	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, foo); err != nil {
		t.Fatal(err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &jsonData); err != nil {
		t.Fatal(err)
	}
	payload := jsonData["data"].(map[string]interface{})

	// Verify that empty ID of type string gets omitted. See:
	// https://github.com/google/jsonapi/issues/83#issuecomment-285611425
	_, ok := payload["id"]
	if ok {
		t.Fatal("Was expecting the data.id member to be omitted")
	}
}

func TestMarshall_invalidIDType(t *testing.T) {
	type badIDStruct struct {
		ID *bool `jsonapi:"primary,cars"`
	}
	id := true
	o := &badIDStruct{ID: &id}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, o); err != ErrBadJSONAPIID {
		t.Fatalf(
			"Was expecting a `%s` error, got `%s`", ErrBadJSONAPIID, err,
		)
	}
}

func TestOmitsEmptyAnnotation(t *testing.T) {
	book := &Book{
		Author:      "aren55555",
		PublishedAt: time.Now().AddDate(0, -1, 0),
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, book); err != nil {
		t.Fatal(err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &jsonData); err != nil {
		t.Fatal(err)
	}
	attributes := jsonData["data"].(map[string]interface{})["attributes"].(map[string]interface{})

	// Verify that the specifically omitted field were omitted
	if val, exists := attributes["title"]; exists {
		t.Fatalf("Was expecting the data.attributes.title key/value to have been omitted - it was not and had a value of %v", val)
	}
	if val, exists := attributes["pages"]; exists {
		t.Fatalf("Was expecting the data.attributes.pages key/value to have been omitted - it was not and had a value of %v", val)
	}

	// Verify the implicitly omitted fields were omitted
	if val, exists := attributes["PublishedAt"]; exists {
		t.Fatalf("Was expecting the data.attributes.PublishedAt key/value to have been implicitly omitted - it was not and had a value of %v", val)
	}

	// Verify the unset fields were not omitted
	if _, exists := attributes["isbn"]; !exists {
		t.Fatal("Was expecting the data.attributes.isbn key/value to have NOT been omitted")
	}
}

func TestHasPrimaryAnnotation(t *testing.T) {
	testModel := &Blog{
		ID:        5,
		Title:     "Title 1",
		CreatedAt: time.Now(),
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, testModel); err != nil {
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
		t.Fatalf("ID not transferred")
	}
}

func TestSupportsAttributes(t *testing.T) {
	testModel := &Blog{
		ID:        5,
		Title:     "Title 1",
		CreatedAt: time.Now(),
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, testModel); err != nil {
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
	if err := MarshalPayload(out, testModel); err != nil {
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

func TestMarshal_Times(t *testing.T) {
	aTime := time.Date(2016, 8, 17, 8, 27, 12, 23849, time.UTC)

	for _, tc := range []struct {
		desc         string
		input        *TimestampModel
		verification func(data map[string]interface{}) error
	}{
		{
			desc: "default_byValue",
			input: &TimestampModel{
				ID:       5,
				DefaultV: aTime,
			},
			verification: func(root map[string]interface{}) error {
				v := root["data"].(map[string]interface{})["attributes"].(map[string]interface{})["defaultv"].(float64)
				if got, want := int64(v), aTime.Unix(); got != want {
					return fmt.Errorf("got %v, want %v", got, want)
				}
				return nil
			},
		},
		{
			desc: "default_byPointer",
			input: &TimestampModel{
				ID:       5,
				DefaultP: &aTime,
			},
			verification: func(root map[string]interface{}) error {
				v := root["data"].(map[string]interface{})["attributes"].(map[string]interface{})["defaultp"].(float64)
				if got, want := int64(v), aTime.Unix(); got != want {
					return fmt.Errorf("got %v, want %v", got, want)
				}
				return nil
			},
		},
		{
			desc: "iso8601_byValue",
			input: &TimestampModel{
				ID:       5,
				ISO8601V: aTime,
			},
			verification: func(root map[string]interface{}) error {
				v := root["data"].(map[string]interface{})["attributes"].(map[string]interface{})["iso8601v"].(string)
				if got, want := v, aTime.UTC().Format(iso8601TimeFormat); got != want {
					return fmt.Errorf("got %v, want %v", got, want)
				}
				return nil
			},
		},
		{
			desc: "iso8601_byPointer",
			input: &TimestampModel{
				ID:       5,
				ISO8601P: &aTime,
			},
			verification: func(root map[string]interface{}) error {
				v := root["data"].(map[string]interface{})["attributes"].(map[string]interface{})["iso8601p"].(string)
				if got, want := v, aTime.UTC().Format(iso8601TimeFormat); got != want {
					return fmt.Errorf("got %v, want %v", got, want)
				}
				return nil
			},
		},
		{
			desc: "rfc3339_byValue",
			input: &TimestampModel{
				ID:       5,
				RFC3339V: aTime,
			},
			verification: func(root map[string]interface{}) error {
				v := root["data"].(map[string]interface{})["attributes"].(map[string]interface{})["rfc3339v"].(string)
				if got, want := v, aTime.UTC().Format(time.RFC3339); got != want {
					return fmt.Errorf("got %v, want %v", got, want)
				}
				return nil
			},
		},
		{
			desc: "rfc3339_byPointer",
			input: &TimestampModel{
				ID:       5,
				RFC3339P: &aTime,
			},
			verification: func(root map[string]interface{}) error {
				v := root["data"].(map[string]interface{})["attributes"].(map[string]interface{})["rfc3339p"].(string)
				if got, want := v, aTime.UTC().Format(time.RFC3339); got != want {
					return fmt.Errorf("got %v, want %v", got, want)
				}
				return nil
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			out := bytes.NewBuffer(nil)
			if err := MarshalPayload(out, tc.input); err != nil {
				t.Fatal(err)
			}
			// Use the standard JSON library to traverse the genereated JSON payload.
			data := map[string]interface{}{}
			json.Unmarshal(out.Bytes(), &data)
			if tc.verification != nil {
				if err := tc.verification(data); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestSupportsLinkable(t *testing.T) {
	testModel := &Blog{
		ID:        5,
		Title:     "Title 1",
		CreatedAt: time.Now(),
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, testModel); err != nil {
		t.Fatal(err)
	}

	resp := new(OnePayload)
	if err := json.NewDecoder(out).Decode(resp); err != nil {
		t.Fatal(err)
	}

	data := resp.Data

	if data.Links == nil {
		t.Fatal("Expected data.links")
	}
	links := *data.Links

	self, hasSelf := links["self"]
	if !hasSelf {
		t.Fatal("Expected 'self' link to be present")
	}
	if _, isString := self.(string); !isString {
		t.Fatal("Expected 'self' to contain a string")
	}

	comments, hasComments := links["comments"]
	if !hasComments {
		t.Fatal("expect 'comments' to be present")
	}
	commentsMap, isMap := comments.(map[string]interface{})
	if !isMap {
		t.Fatal("Expected 'comments' to contain a map")
	}

	commentsHref, hasHref := commentsMap["href"]
	if !hasHref {
		t.Fatal("Expect 'comments' to contain an 'href' key/value")
	}
	if _, isString := commentsHref.(string); !isString {
		t.Fatal("Expected 'href' to contain a string")
	}

	commentsMeta, hasMeta := commentsMap["meta"]
	if !hasMeta {
		t.Fatal("Expect 'comments' to contain a 'meta' key/value")
	}
	commentsMetaMap, isMap := commentsMeta.(map[string]interface{})
	if !isMap {
		t.Fatal("Expected 'comments' to contain a map")
	}

	commentsMetaObject := Meta(commentsMetaMap)
	countsMap, isMap := commentsMetaObject["counts"].(map[string]interface{})
	if !isMap {
		t.Fatal("Expected 'counts' to contain a map")
	}
	for k, v := range countsMap {
		if _, isNum := v.(float64); !isNum {
			t.Fatalf("Exepected value at '%s' to be a numeric (float64)", k)
		}
	}
}

func TestInvalidLinkable(t *testing.T) {
	testModel := &BadComment{
		ID:   5,
		Body: "Hello World",
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, testModel); err == nil {
		t.Fatal("Was expecting an error")
	}
}

func TestSupportsMetable(t *testing.T) {
	testModel := &Blog{
		ID:        5,
		Title:     "Title 1",
		CreatedAt: time.Now(),
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, testModel); err != nil {
		t.Fatal(err)
	}

	resp := new(OnePayload)
	if err := json.NewDecoder(out).Decode(resp); err != nil {
		t.Fatal(err)
	}

	data := resp.Data
	if data.Meta == nil {
		t.Fatalf("Expected data.meta")
	}

	meta := Meta(*data.Meta)
	if e, a := "extra details regarding the blog", meta["detail"]; e != a {
		t.Fatalf("Was expecting meta.detail to be %q, got %q", e, a)
	}
}

func TestRelations(t *testing.T) {
	testModel := testBlog()

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, testModel); err != nil {
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
	} else {
		if relations["posts"].(map[string]interface{})["links"] == nil {
			t.Fatalf("Posts relationship links were not materialized")
		}
		if relations["posts"].(map[string]interface{})["meta"] == nil {
			t.Fatalf("Posts relationship meta were not materialized")
		}
	}

	if relations["current_post"] == nil {
		t.Fatalf("Current post relationship was not materialized")
	} else {
		if relations["current_post"].(map[string]interface{})["links"] == nil {
			t.Fatalf("Current post relationship links were not materialized")
		}
		if relations["current_post"].(map[string]interface{})["meta"] == nil {
			t.Fatalf("Current post relationship meta were not materialized")
		}
	}

	if len(relations["posts"].(map[string]interface{})["data"].([]interface{})) != 2 {
		t.Fatalf("Did not materialize two posts")
	}
}

func TestNoRelations(t *testing.T) {
	testModel := &Blog{ID: 1, Title: "Title 1", CreatedAt: time.Now()}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, testModel); err != nil {
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

func TestMarshalPayloadWithoutIncluded(t *testing.T) {
	data := &Post{
		ID:       1,
		BlogID:   2,
		ClientID: "123e4567-e89b-12d3-a456-426655440000",
		Title:    "Foo",
		Body:     "Bar",
		Comments: []*Comment{
			{
				ID:   20,
				Body: "First",
			},
			{
				ID:   21,
				Body: "Hello World",
			},
		},
		LatestComment: &Comment{
			ID:   22,
			Body: "Cool!",
		},
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayloadWithoutIncluded(out, data); err != nil {
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

func TestMarshalPayload_many(t *testing.T) {
	data := []interface{}{
		&Blog{
			ID:        5,
			Title:     "Title 1",
			CreatedAt: time.Now(),
			Posts: []*Post{
				{
					ID:    1,
					Title: "Foo",
					Body:  "Bar",
				},
				{
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
				{
					ID:    3,
					Title: "Foo",
					Body:  "Bar",
				},
				{
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
	if err := MarshalPayload(out, data); err != nil {
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

func TestMarshalMany_WithSliceOfStructPointers(t *testing.T) {
	var data []*Blog
	for len(data) < 2 {
		data = append(data, testBlog())
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, data); err != nil {
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

func TestMarshalManyWithoutIncluded(t *testing.T) {
	var data []*Blog
	for len(data) < 2 {
		data = append(data, testBlog())
	}

	out := bytes.NewBuffer(nil)
	if err := MarshalPayloadWithoutIncluded(out, data); err != nil {
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

	if resp.Included != nil {
		t.Fatalf("Encoding json response did not omit included")
	}
}

func TestMarshalMany_SliceOfInterfaceAndSliceOfStructsSameJSON(t *testing.T) {
	structs := []*Book{
		{ID: 1, Author: "aren55555", ISBN: "abc"},
		{ID: 2, Author: "shwoodard", ISBN: "xyz"},
	}
	interfaces := []interface{}{}
	for _, s := range structs {
		interfaces = append(interfaces, s)
	}

	// Perform Marshals
	structsOut := new(bytes.Buffer)
	if err := MarshalPayload(structsOut, structs); err != nil {
		t.Fatal(err)
	}
	interfacesOut := new(bytes.Buffer)
	if err := MarshalPayload(interfacesOut, interfaces); err != nil {
		t.Fatal(err)
	}

	// Generic JSON Unmarshal
	structsData, interfacesData :=
		make(map[string]interface{}), make(map[string]interface{})
	if err := json.Unmarshal(structsOut.Bytes(), &structsData); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(interfacesOut.Bytes(), &interfacesData); err != nil {
		t.Fatal(err)
	}

	// Compare Result
	if !reflect.DeepEqual(structsData, interfacesData) {
		t.Fatal("Was expecting the JSON API generated to be the same")
	}
}

func TestMarshal_InvalidIntefaceArgument(t *testing.T) {
	out := new(bytes.Buffer)
	if err := MarshalPayload(out, true); err != ErrUnexpectedType {
		t.Fatal("Was expecting an error")
	}
	if err := MarshalPayload(out, 25); err != ErrUnexpectedType {
		t.Fatal("Was expecting an error")
	}
	if err := MarshalPayload(out, Book{}); err != ErrUnexpectedType {
		t.Fatal("Was expecting an error")
	}
}

func TestMarshal_EmptyAttributesNotIncluded(t *testing.T) {
	type Primary struct {
		ID          string     `jsonapi:"primary,primary"`
		Attr        string     `jsonapi:"attr,attr,omitempty"`
		Secondary   *Primary   `jsonapi:"relation,secondary,omitempty"`
		Secondaries []*Primary `jsonapi:"relation,secondaries,omitempty"`
	}

	p := &Primary{
		ID:   "1",
		Attr: "a",
		Secondary: &Primary{
			ID: "2",
		},
		Secondaries: []*Primary{
			{ID: "3"},
		},
	}

	t.Run("Single payload", func(t *testing.T) {
		out := bytes.NewBuffer(nil)
		if err := MarshalPayload(out, p); err != nil {
			t.Fatal(err)
		}

		var jsonData map[string]interface{}
		if err := json.Unmarshal(out.Bytes(), &jsonData); err != nil {
			t.Fatal(err)
		}

		_, ok := jsonData["included"].([]interface{})
		if ok {
			t.Fatal("Was expecting included to be empty")
		}
	})

	t.Run("Many payload", func(t *testing.T) {

		out := bytes.NewBuffer(nil)
		if err := MarshalPayload(out, []*Primary{p}); err != nil {
			t.Fatal(err)
		}

		var jsonData map[string]interface{}
		if err := json.Unmarshal(out.Bytes(), &jsonData); err != nil {
			t.Fatal(err)
		}

		_, ok := jsonData["included"].([]interface{})
		if ok {
			t.Fatal("Was expecting included to be empty")
		}
	})
}

func testBlog() *Blog {
	return &Blog{
		ID:        5,
		Title:     "Title 1",
		CreatedAt: time.Now(),
		Posts: []*Post{
			{
				ID:    1,
				Title: "Foo",
				Body:  "Bar",
				Comments: []*Comment{
					{
						ID:   1,
						Body: "foo",
					},
					{
						ID:   2,
						Body: "bar",
					},
				},
				LatestComment: &Comment{
					ID:   1,
					Body: "foo",
				},
			},
			{
				ID:    2,
				Title: "Fuubar",
				Body:  "Bas",
				Comments: []*Comment{
					{
						ID:   1,
						Body: "foo",
					},
					{
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
				{
					ID:   1,
					Body: "foo",
				},
				{
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
