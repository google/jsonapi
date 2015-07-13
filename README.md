# jsonapi

A serailizer/deserializer for json payloads that comply to the
[jsonapi.org](http://jsonapi.org) spec in go.

## Background

You are working in your Go web application and you have a struct that is
similar to how your datbase table looks.  You need to send and receive
json payloads that adhere jsonapi spec.  Once you realized that your
json needed to take on this special form, you went down the path of
creating more structs to be able to serialize and deserialize jsonapi
payloads.  Then more models required these additional structure.  Ugh!
In comes jsonapi.  You can keep your model structs as is and use struct
field tags to indicate to jsonapi how you want your response built or
your request deserialzied.  What about my relationships?  jsonapi
supports relationships out of the box and will even side load them in
your response into an "included" array--that contains associated
objects.

## Introduction

jsonapi uses StructField tags to annotate the structs fields that you
already have and use in your app and then reads and writes jsonapi.org
output based on the instructions you give the library in your jsonapi
tags.  Let's take an example.  In your app,
you most likely have structs that look similar to these,


```go
type Blog struct {
	Id            int       `json:"id"`
	Title         string    `json:"title"`
	Posts         []*Post   `json:"posts"`
	CurrentPost   *Post     `json:"current_post"`
	CurrentPostId int       `json:"current_post_id"`
	CreatedAt     time.Time `json:"created_at"`
	ViewCount     int       `json:"view_count"`
}

type Post struct {
	Id       int        `json:"id"`
	BlogId   int        `json:"blog_id"`
	Title    string     `json:"title"`
	Body     string     `json:"body"`
	Comments []*Comment `json:"comments"`
}

type Comment struct {
	Id     int    `json:"id"`
	PostId int    `json:"post_id"`
	Body   string `json:"body"`
}
```

These structs may or may not resemble the layout of your database.  But
these are the ones that you want to use right?  You wouldn't want to use
structs like those that jsonapi sends because it is very hard to get at all of
your data easily.


## Tags Example

You want jsonapi.org style inputs and ouputs but you want to keep your
structs that you already have.  Use the jsonapi lib with the "jsonapi"
tag on your struct fields along with its Marshal and Unmarshal methods
to construct and read your responses and replies, respectively.  Here's
an example of the structs above using jsonapi tags,

```go
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
```

## Handler Examples

Now you have your structs are prepared to be seralized or materialized.
What about the rest?

### Create

You can Unmarshal a jsonapi payload using `jsonapi.UnmarshalPayload`; convert an io
into a struct instance using jsonapi tags on struct fields.  Method supports single
request payloads only, at the moment. Bulk creates and updates are not supported yet.

#### `UnmarshalPayload`

```go
UnmarshalPayload(in io.Reader, model interface{})
```

Visit [godoc](http://godoc.org/github.com/shwoodard/jsonapi#UnmarshalPayload)

#### `MarshalOnePayload`

This method encodes a response for a single record only. If you want to serialize many
records, see, [MarshalManyPayload](#marshalmanypayload). Wrties a jsonapi response, with
related records sideloaded, into `included` array.

```go
MarshalOnePayloadEmbedded(w io.Writer, model interface{}) error
```

Visit [godoc](http://godoc.org/github.com/shwoodard/jsonapi#MarshalOnePayload)

#### Example

```go
func CreateBlog(w http.ResponseWriter, r *http.Request) {
	blog := new(Blog)

	if err := jsonapi.UnmarshalPayload(r.Body, blog); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// ...do stuff with your blog...

	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/vnd.api+json")

	if err := jsonapi.MarshalOnePayload(w, blog); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
```

### List
#### `MarshalManyPayload`

```go
MarshalManyPayload(w io.Writer, models []interface{}) error
```

Visit [godoc](http://godoc.org/github.com/shwoodard/jsonapi#MashalManyPayload)

#### Example

```go
func ListBlogs(w http.ResponseWriter, r *http.Request) {
	// ... fetch your blogs and filter, offset, limit, etc ...

	blogs := testBlogsForList()

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/vnd.api+json")
	if err := jsonapi.MarshalManyPayload(w, blogs); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
```
