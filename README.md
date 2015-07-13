# jsonapi

[![Build Status](https://travis-ci.org/shwoodard/jsonapi.svg?branch=master)](https://travis-ci.org/shwoodard/jsonapi)

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


## Tags

### Example

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

### Reference

The **jsonapi** Tag Reference

#### `primary`

```
`jsonapi:"primary,<type field output>"`
```

This indicates that this is the primary key field for this struct type. Tag
value arguments are comma separated.  The first argument must be, `primary`, and
the second must be the name that should appear in the `type` field for all data
objects that represent this type of model.

#### `attr`

```
`jsonapi:"attr,<key name in attributes hash>"`
```

These fields' values should end up in the `attributes`hash for a record.  The first
argument must be, `attr`, and the second should be the name for the key to display in
the the `attributes` hash for that record.

#### `relation`

```
`jsonapi:"relation,<key name in relationships hash>"`
```

Relations are struct fields that represent a one-to-one or one-to-many to other structs.
jsonapi will traverse the graph of relationships and marshal or unmarshal records.  The first
argument must be, `relation`, and the second should be the name of the relationship, used as
the key in the `relationships` hash for the record.

## Methods Reference

**All `Marshal` and `Unmarshal` methods expect pointers to struct
instance or slices of the same contained with the `interface{}`s**

Now you have your structs are prepared to be seralized or materialized.
What about the rest?

### Create Record Example

You can Unmarshal a jsonapi payload using `jsonapi.UnmarshalPayload`; convert an io
into a struct instance using jsonapi tags on struct fields.  Method supports single
request payloads only, at the moment. Bulk creates and updates are not supported yet.

#### `UnmarshalPayload`

```go
UnmarshalPayload(in io.Reader, model interface{})
```

Visit [godoc](http://godoc.org/github.com/shwoodard/jsonapi#UnmarshalPayload)

#### `MarshalOnePayload`

```go
MarshalOnePayload(w io.Writer, model interface{}) error
```

Visit [godoc](http://godoc.org/github.com/shwoodard/jsonapi#MarshalOnePayload)

This method encodes a response for a single record only. If you want to serialize many
records, see, [MarshalManyPayload](#marshalmanypayload). Wrties a jsonapi response, with
related records sideloaded, into `included` array.

#### Handler Exmaple Code

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

### List Records Example

#### `MarshalManyPayload`

```go
MarshalManyPayload(w io.Writer, models []interface{}) error
```

Visit [godoc](http://godoc.org/github.com/shwoodard/jsonapi#MashalManyPayload)

Takes an `io.Writer` and an slice of `interface{}`.  Note, if you have a
type safe array of your structs, like,

```go
var blogs []*Blog
```

you will need to interate over the slice of `Blog` pointers and append
them to an interface array, like,

```go
blogInterface := make([]interface{}, len(blogs))

for i, blog := range blogs {
  blogInterface[i] = blog
}

```

Alternatively, you can insert your `Blog`s into a slice of `interface{}`
the first time.  For example when you fetch the `Blog`s from the db
`append` them to an `[]interface{}` rather than a `[]*Blog`,

```go
FetchBlogs() ([]interface{}, error)
```

#### Handler Example Code

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

## Testing

### `MarshalOnePayloadEmbedded`

```go
MarshalOnePayloadEmbedded(w io.Writer, model interface{}) error
```

Visit [godoc](http://godoc.org/github.com/shwoodard/jsonapi#MarshalOnePayloadEmbedded)

This method not meant to for use in implementation code, although feel
free.  This method was created for use in tests.  In most cases, your
request payloads for create will be embedded rather than sideloaded for related records.
This method will serialize a single struct pointer into an embedded json
response.  In other words, there will be no, "included", array in the json
all relationships will be serailized inline in the data.

However, in tests, you may want to construct payloads to post to create methods
that are embedded to most closely resember the payloads that will be produced by
the client.  This is what this method is intended for.

model interface{} should be a pointer to a struct.

### Example

```go
out := bytes.NewBuffer(nil)

// testModel returns a pointer to a Blog
jsonapi.MarshalOnePayloadEmbedded(out, testModel())

h := new(BlogsHandler)

w := httptest.NewRecorder()
r, _ := http.NewRequest("POST", "/blogs", out)

h.CreateBlog(w, r)

blog := new(Blog)
jsonapi.UnmarshalPayload(w.Body, blog)

// ... assert stuff about blog here ...
```

## Contributing

Fork, Change, Pull Request *with tests*.
