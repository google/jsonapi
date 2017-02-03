package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/google/jsonapi"
)

func createBlog(w http.ResponseWriter, r *http.Request) {
	jsonapiRuntime := jsonapi.NewRuntime().Instrument("blogs.create")

	blog := new(Blog)

	if err := jsonapiRuntime.UnmarshalPayload(r.Body, blog); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// ...do stuff with your blog...

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", jsonapi.MediaType)

	if err := jsonapiRuntime.MarshalOnePayload(w, blog); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func listBlogs(w http.ResponseWriter, r *http.Request) {
	jsonapiRuntime := jsonapi.NewRuntime().Instrument("blogs.list")
	// ...fetch your blogs, filter, offset, limit, etc...

	// but, for now
	blogs := testBlogsForList()

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", jsonapi.MediaType)
	if err := jsonapiRuntime.MarshalManyPayload(w, blogs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func showBlog(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")

	// ...fetch your blog...

	intID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonapiRuntime := jsonapi.NewRuntime().Instrument("blogs.show")

	// but, for now
	blog := testBlogForCreate(intID)
	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", jsonapi.MediaType)
	if err := jsonapiRuntime.MarshalOnePayload(w, blog); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func echoBlogs(w http.ResponseWriter, r *http.Request) {
	jsonapiRuntime := jsonapi.NewRuntime().Instrument("blogs.echo")

	// Fetch the blogs from the HTTP request body
	data, err := jsonapiRuntime.UnmarshalManyPayload(r.Body, reflect.TypeOf(new(Blog)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// Type assert the []interface{} to []*Blog
	blogs := []*Blog{}
	for _, b := range data {
		blog, ok := b.(*Blog)
		if !ok {
			http.Error(w, "Unexpected type", http.StatusInternalServerError)
		}
		blogs = append(blogs, blog)
	}

	// Echo the blogs to the response body
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", jsonapi.MediaType)
	if err := jsonapiRuntime.MarshalManyPayload(w, blogs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	jsonapi.Instrumentation = func(r *jsonapi.Runtime, eventType jsonapi.Event, callGUID string, dur time.Duration) {
		metricPrefix := r.Value("instrument").(string)

		if eventType == jsonapi.UnmarshalStart {
			fmt.Printf("%s: id, %s, started at %v\n", metricPrefix+".jsonapi_unmarshal_time", callGUID, time.Now())
		}

		if eventType == jsonapi.UnmarshalStop {
			fmt.Printf("%s: id, %s, stopped at, %v , and took %v to unmarshal payload\n", metricPrefix+".jsonapi_unmarshal_time", callGUID, time.Now(), dur)
		}

		if eventType == jsonapi.MarshalStart {
			fmt.Printf("%s: id, %s, started at %v\n", metricPrefix+".jsonapi_marshal_time", callGUID, time.Now())
		}

		if eventType == jsonapi.MarshalStop {
			fmt.Printf("%s: id, %s, stopped at, %v , and took %v to marshal payload\n", metricPrefix+".jsonapi_marshal_time", callGUID, time.Now(), dur)
		}
	}

	http.HandleFunc("/blogs", func(w http.ResponseWriter, r *http.Request) {
		if !regexp.MustCompile(`application/vnd\.api\+json`).Match([]byte(r.Header.Get("Accept"))) {
			http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
			return
		}

		if r.Method == http.MethodPost {
			createBlog(w, r)
		} else if r.Method == http.MethodPut {
			echoBlogs(w, r)
		} else if r.FormValue("id") != "" {
			showBlog(w, r)
		} else {
			listBlogs(w, r)
		}
	})

	exerciseHandler()
}

func testBlogForCreate(i int) *Blog {
	return &Blog{
		ID:        1 * i,
		Title:     "Title 1",
		CreatedAt: time.Now(),
		Posts: []*Post{
			&Post{
				ID:    1 * i,
				Title: "Foo",
				Body:  "Bar",
				Comments: []*Comment{
					&Comment{
						ID:   1 * i,
						Body: "foo",
					},
					&Comment{
						ID:   2 * i,
						Body: "bar",
					},
				},
			},
			&Post{
				ID:    2 * i,
				Title: "Fuubar",
				Body:  "Bas",
				Comments: []*Comment{
					&Comment{
						ID:   1 * i,
						Body: "foo",
					},
					&Comment{
						ID:   3 * i,
						Body: "bas",
					},
				},
			},
		},
		CurrentPost: &Post{
			ID:    1 * i,
			Title: "Foo",
			Body:  "Bar",
			Comments: []*Comment{
				&Comment{
					ID:   1 * i,
					Body: "foo",
				},
				&Comment{
					ID:   2 * i,
					Body: "bar",
				},
			},
		},
	}
}

func testBlogsForList() []interface{} {
	blogs := make([]interface{}, 0, 10)

	for i := 0; i < 10; i++ {
		blogs = append(blogs, testBlogForCreate(i))
	}

	return blogs
}

func exerciseHandler() {
	// list
	req, _ := http.NewRequest(http.MethodGet, "/blogs", nil)

	req.Header.Set("Accept", jsonapi.MediaType)

	w := httptest.NewRecorder()

	fmt.Println("============ start list ===========")
	http.DefaultServeMux.ServeHTTP(w, req)
	fmt.Println("============ stop list ===========")

	jsonReply, _ := ioutil.ReadAll(w.Body)

	fmt.Println("============ jsonapi response from list ===========")
	fmt.Println(string(jsonReply))
	fmt.Println("============== end raw jsonapi from list =============")

	// show
	req, _ = http.NewRequest(http.MethodGet, "/blogs?id=1", nil)

	req.Header.Set("Accept", jsonapi.MediaType)

	w = httptest.NewRecorder()

	fmt.Println("============ start show ===========")
	http.DefaultServeMux.ServeHTTP(w, req)
	fmt.Println("============ stop show ===========")

	jsonReply, _ = ioutil.ReadAll(w.Body)

	fmt.Println("============ jsonapi response from show ===========")
	fmt.Println(string(jsonReply))
	fmt.Println("============== end raw jsonapi from show =============")

	// create
	blog := testBlogForCreate(1)
	in := bytes.NewBuffer(nil)
	jsonapi.MarshalOnePayloadEmbedded(in, blog)

	req, _ = http.NewRequest(http.MethodPost, "/blogs", in)

	req.Header.Set("Accept", jsonapi.MediaType)

	w = httptest.NewRecorder()

	fmt.Println("============ start create ===========")
	http.DefaultServeMux.ServeHTTP(w, req)
	fmt.Println("============ stop create ===========")

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, w.Body)

	fmt.Println("============ jsonapi response from create ===========")
	fmt.Println(buf.String())
	fmt.Println("============== end raw jsonapi response =============")

	// echo
	blogs := []interface{}{
		testBlogForCreate(1),
		testBlogForCreate(2),
		testBlogForCreate(3),
	}
	in = bytes.NewBuffer(nil)
	jsonapi.MarshalManyPayload(in, blogs)

	req, _ = http.NewRequest(http.MethodPut, "/blogs", in)

	req.Header.Set("Accept", jsonapi.MediaType)

	w = httptest.NewRecorder()

	fmt.Println("============ start echo ===========")
	http.DefaultServeMux.ServeHTTP(w, req)
	fmt.Println("============ stop echo ===========")

	buf = bytes.NewBuffer(nil)
	io.Copy(buf, w.Body)

	fmt.Println("============ jsonapi response from create ===========")
	fmt.Println(buf.String())
	fmt.Println("============== end raw jsonapi response =============")

	responseBlog := new(Blog)

	jsonapi.UnmarshalPayload(buf, responseBlog)

	out := bytes.NewBuffer(nil)
	json.NewEncoder(out).Encode(responseBlog)

	fmt.Println("================ Viola! Converted back our Blog struct =================")
	fmt.Println(string(out.Bytes()))
	fmt.Println("================ end marshal materialized Blog struct =================")
}

type Blog struct {
	ID            int       `jsonapi:"primary,blogs"`
	Title         string    `jsonapi:"attr,title"`
	Posts         []*Post   `jsonapi:"relation,posts"`
	CurrentPost   *Post     `jsonapi:"relation,current_post"`
	CurrentPostID int       `jsonapi:"attr,current_post_id"`
	CreatedAt     time.Time `jsonapi:"attr,created_at"`
	ViewCount     int       `jsonapi:"attr,view_count"`
}

type Post struct {
	ID       int        `jsonapi:"primary,posts"`
	BlogID   int        `jsonapi:"attr,blog_id"`
	Title    string     `jsonapi:"attr,title"`
	Body     string     `jsonapi:"attr,body"`
	Comments []*Comment `jsonapi:"relation,comments"`
}

type Comment struct {
	ID     int    `jsonapi:"primary,comments"`
	PostID int    `jsonapi:"attr,post_id"`
	Body   string `jsonapi:"attr,body"`
}

// Blog Links
func (blog Blog) JSONAPILinks() *map[string]interface{} {
	return &map[string]interface{}{
		"self": fmt.Sprintf("https://example.com/blogs/%d", blog.ID),
	}
}

func (blog Blog) JSONAPIRelationshipLinks(relation string) *map[string]interface{} {
	if relation == "posts" {
		return &map[string]interface{}{
			"related": fmt.Sprintf("https://example.com/blogs/%d/posts", blog.ID),
		}
	}
	if relation == "current_post" {
		return &map[string]interface{}{
			"related": fmt.Sprintf("https://example.com/blogs/%d/current_post", blog.ID),
		}
	}
	return nil
}

// Blog Meta
func (blog Blog) JSONAPIMeta() map[string]interface{} {
	return map[string]interface{}{
		"detail": "extra details regarding the blog",
	}
}

func (blog Blog) JSONAPIRelationshipMeta(relation string) map[string]interface{} {
	if relation == "posts" {
		return map[string]interface{}{
			"detail": "posts meta information",
		}
	}
	if relation == "current_post" {
		return map[string]interface{}{
			"detail": "current post meta information",
		}
	}
	return nil
}
