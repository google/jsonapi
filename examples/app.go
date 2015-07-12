package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"time"

	"github.com/shwoodard/jsonapi"
)

func createBlog(w http.ResponseWriter, r *http.Request) {
	blog := new(Blog)

	if err := jsonapi.UnmarshalPayload(r.Body, blog); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// ...do stuff with your blog ...

	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/vnd.api+json")

	if err := jsonapi.MarshalOnePayload(w, blog); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func listBlogs(w http.ResponseWriter, r *http.Request) {
	// ... fetch your blogs and filter, offset, limit, etc ...

	blogs := testBlogsForList()

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/vnd.api+json")
	if err := jsonapi.MarshalManyPayload(w, blogs); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func main() {
	http.HandleFunc("/blogs", func(w http.ResponseWriter, r *http.Request) {
		if !regexp.MustCompile(`application/vnd\.api\+json`).Match([]byte(r.Header.Get("Accept"))) {
			http.Error(w, "Not Acceptable", 406)
			return
		}

		if r.Method == "POST" {
			createBlog(w, r)
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

func testBlogsForList() Blogs {
	blogs := make(Blogs, 0, 10)

	for i := 0; i < 10; i += 1 {
		blogs = append(blogs, testBlogForCreate(i))
	}

	return blogs
}

func exerciseHandler() {
	req, _ := http.NewRequest("GET", "/blogs", nil)

	req.Header.Set("Accept", "application/vnd.api+json")

	w := httptest.NewRecorder()

	http.DefaultServeMux.ServeHTTP(w, req)

	buf := new(bytes.Buffer)
	io.Copy(buf, w.Body)

	fmt.Println("============ jsonapi response from list ===========\n")
	fmt.Println(buf.String())
	fmt.Println("============== end raw jsonapi from list =============")

	blog := testBlogForCreate(1)
	in := bytes.NewBuffer(nil)
	jsonapi.MarshalOnePayloadEmbedded(in, blog)

	req, _ = http.NewRequest("POST", "/blogs", in)

	req.Header.Set("Accept", "application/vnd.api+json")

	w = httptest.NewRecorder()

	http.DefaultServeMux.ServeHTTP(w, req)

	buf = new(bytes.Buffer)
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

type Blogs []*Blog

func (b Blogs) GetData() []interface{} {
	d := make([]interface{}, len(b))
	for i, blog := range b {
		d[i] = blog
	}
	return d
}
