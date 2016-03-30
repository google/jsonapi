package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"time"

	"github.com/shwoodard/jsonapi"
)

func createBlog(w http.ResponseWriter, r *http.Request) {
	jsonapiRuntime := jsonapi.NewRuntime().Instrument("blogs.create")

	blog := new(Blog)

	if err := jsonapiRuntime.UnmarshalPayload(r.Body, blog); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// ...do stuff with your blog...

	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/vnd.api+json")

	if err := jsonapiRuntime.MarshalOnePayload(w, blog); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func listBlogs(w http.ResponseWriter, r *http.Request) {
	jsonapiRuntime := jsonapi.NewRuntime().Instrument("blogs.list")
	// ...fetch your blogs, filter, offset, limit, etc...

	// but, for now
	blogs := testBlogsForList()

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/vnd.api+json")
	if err := jsonapiRuntime.MarshalManyPayload(w, blogs); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func showBlog(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")

	// ...fetch your blog...

	intId, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	jsonapiRuntime := jsonapi.NewRuntime().Instrument("blogs.show")

	// but, for now
	blog := testBlogForCreate(intId)
	w.WriteHeader(200)

	w.Header().Set("Content-Type", "application/vnd.api+json")
	if err := jsonapiRuntime.MarshalOnePayload(w, blog); err != nil {
		http.Error(w, err.Error(), 500)
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
			http.Error(w, "Unsupported Media Type", 415)
			return
		}

		if r.Method == "POST" {
			createBlog(w, r)
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
		Id:        1 * i,
		Title:     "Title 1",
		CreatedAt: time.Now(),
		Posts: []*Post{
			&Post{
				Id:    1 * i,
				Title: "Foo",
				Body:  "Bar",
				Comments: []*Comment{
					&Comment{
						Id:   1 * i,
						Body: "foo",
					},
					&Comment{
						Id:   2 * i,
						Body: "bar",
					},
				},
			},
			&Post{
				Id:    2 * i,
				Title: "Fuubar",
				Body:  "Bas",
				Comments: []*Comment{
					&Comment{
						Id:   1 * i,
						Body: "foo",
					},
					&Comment{
						Id:   3 * i,
						Body: "bas",
					},
				},
			},
		},
		CurrentPost: &Post{
			Id:    1 * i,
			Title: "Foo",
			Body:  "Bar",
			Comments: []*Comment{
				&Comment{
					Id:   1 * i,
					Body: "foo",
				},
				&Comment{
					Id:   2 * i,
					Body: "bar",
				},
			},
		},
	}
}

func testBlogsForList() []interface{} {
	blogs := make([]interface{}, 0, 10)

	for i := 0; i < 10; i += 1 {
		blogs = append(blogs, testBlogForCreate(i))
	}

	return blogs
}

func exerciseHandler() {
	// list
	req, _ := http.NewRequest("GET", "/blogs", nil)

	req.Header.Set("Accept", "application/vnd.api+json")

	w := httptest.NewRecorder()

	fmt.Println("============ start list ===========\n")
	http.DefaultServeMux.ServeHTTP(w, req)
	fmt.Println("============ stop list ===========\n")

	jsonReply, _ := ioutil.ReadAll(w.Body)

	fmt.Println("============ jsonapi response from list ===========\n")
	fmt.Println(string(jsonReply))
	fmt.Println("============== end raw jsonapi from list =============")

	// show
	req, _ = http.NewRequest("GET", "/blogs?id=1", nil)

	req.Header.Set("Accept", "application/vnd.api+json")

	w = httptest.NewRecorder()

	fmt.Println("============ start show ===========\n")
	http.DefaultServeMux.ServeHTTP(w, req)
	fmt.Println("============ stop show ===========\n")

	jsonReply, _ = ioutil.ReadAll(w.Body)

	fmt.Println("\n============ jsonapi response from show ===========\n")
	fmt.Println(string(jsonReply))
	fmt.Println("============== end raw jsonapi from show =============")

	// create
	blog := testBlogForCreate(1)
	in := bytes.NewBuffer(nil)
	jsonapi.MarshalOnePayloadEmbedded(in, blog)

	req, _ = http.NewRequest("POST", "/blogs", in)

	req.Header.Set("Accept", "application/vnd.api+json")

	w = httptest.NewRecorder()

	fmt.Println("============ start create ===========\n")
	http.DefaultServeMux.ServeHTTP(w, req)
	fmt.Println("============ stop create ===========\n")

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, w.Body)

	fmt.Println("\n============ jsonapi response from create ===========\n")
	fmt.Println(buf.String())
	fmt.Println("============== end raw jsonapi response =============")

	responseBlog := new(Blog)

	jsonapi.UnmarshalPayload(buf, responseBlog)

	out := bytes.NewBuffer(nil)
	json.NewEncoder(out).Encode(responseBlog)

	fmt.Println("\n================ Viola! Converted back our Blog struct =================\n")
	fmt.Printf("%s\n", out.Bytes())
	fmt.Println("================ end marshal materialized Blog struct =================")
}

type Blog struct {
	Id            int       `jsonapi:"primary,blogs"`
	Title         string    `jsonapi:"attr,title"`
	Posts         []*Post   `jsonapi:"relation,posts"`
	CurrentPost   *Post     `jsonapi:"relation,current_post"`
	CurrentPostId int       `jsonapi:"attr,current_post_id"`
	CreatedAt     time.Time `jsonapi:"attr,created_at"`
	ViewCount     int       `jsonapi:"attr,view_count"`
}

type Post struct {
	Id       int        `jsonapi:"primary,posts"`
	BlogId   int        `jsonapi:"attr,blog_id"`
	Title    string     `jsonapi:"attr,title"`
	Body     string     `jsonapi:"attr,body"`
	Comments []*Comment `jsonapi:"relation,comments"`
}

type Comment struct {
	Id     int    `jsonapi:"primary,comments"`
	PostId int    `jsonapi:"attr,post_id"`
	Body   string `jsonapi:"attr,body"`
}
