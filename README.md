# jsonapi

[![Build Status](https://travis-ci.org/shwoodard/jsonapi.svg?branch=master)](https://travis-ci.org/shwoodard/jsonapi)

A serailizer/deserializer for json payloads that comply to the
[jsonapi.org](http://jsonapi.org) spec in go.

Also visit, [Godoc](http://godoc.org/github.com/shwoodard/jsonapi).

## Installation

```
go get -u github.com/shwoodard/jsonapi
```

Or, see [Alternative Installation](#alternative-installation).

## Background

You are working in your Go web application and you have a struct that is
organized similarly to how your database schema.  You need to send and
receive json payloads that adhere jsonapi spec.  Once you realize that
your json needed to take on this special form, you go down the path of
creating more structs to be able to serialize and deserialize jsonapi
payloads.  Then there are more models that required these additional
structure.  Ugh! With jsonapi, you can keep your model structs as is and
use [StructTags](http://golang.org/pkg/reflect/#StructTag) to indicate
to jsonapi how you want your response built or your request
deserialized.  What about your relationships?  jsonapi supports
relationships out of the box and will even put them in your response
into an `included` side-loaded slice--that contains associated records.

## Introduction

jsonapi uses [StructField](http://golang.org/pkg/reflect/#StructField)
tags to annotate the structs fields that you already have and use in
your app and then reads and writes [jsonapi.org](http://jsonapi.org)
output based on the instructions you give the library in your jsonapi
tags.  Let's take an example.  In your app, you most likely have structs
that look similar to these,


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
structs like those that jsonapi sends because it is difficult to get at
all of your data easily.

## Example App

[examples/app.go](https://github.com/shwoodard/jsonapi/blob/master/examples/app.go)

This runnable file demonstrates the implementation of a create, a show,
and a list [http.Handler](http://golang.org/pkg/net/http#Handler).  It
outputs some example requests and response as well as serialized
examples of the source/target structs to json.  That is to say, I show
you that the library has successfully taken your jsonapi request and
turned it into your struct types.

To run,

* Make sure you have go installed
* Create the following directories or similar: `~/go`
* `cd` there
* Set `GOPATH` to `PWD` in your shell session, `export GOPATH=$PWD`
* `go get github.com/shoodard/jsonapi`.  (Append `-u` after `get` if you
  are updating.)
* `go run src/github.com/shwoodard/jsonapi/examples/app.go` or `cd
  src/github.com/shwoodard/jsonapi/examples && go run app.go`

## `jsonapi` Tag Reference

### Example

The `jsonapi` [StructTags](http://golang.org/pkg/reflect/#StructTag)
tells this library how to Marshal and Unmarshal your structs into
jsonapi payloads and your jsonapi payloads to structs, respectively.
Then Use jsonapi's Marshal and Unmarshal methods to construct and read
your responses and replies.  Here's an example of the structs above
using jsonapi tags,

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

### Permitted Tag Values

#### `primary`

```
`jsonapi:"primary,<type field output>"`
```

This indicates that this is the primary key field for this struct type.
Tag value arguments are comma separated.  The first argument must be,
`primary`, and the second must be the name that should appear in the
`type`\* field for all data objects that represent this type of model.

\* According the [jsonapi](http://jsonapi.org) spec, the plural record
types are shown in the examples, but not required.

#### `attr`

```
`jsonapi:"attr,<key name in attributes hash>"`
```

These fields' values will end up in the `attributes`hash for a record.
The first argument must be, `attr`, and the second should be the name
for the key to display in the `attributes` hash for that record. The
spec indicates that `attributes` key names should be dasherized for
multiple word field names.

#### `relation`

```
`jsonapi:"relation,<key name in relationships hash>"`
```

Relations are struct fields that represent a one-to-one or one-to-many
relationship with other structs.  jsonapi will traverse the graph of
relationships and Marshal or Unmarshal records.  The first argument must
be, `relation`, and the second should be the name of the relationship,
used as the key in the `relationships` hash for the record.

## Methods Reference

**All `Marshal` and `Unmarshal` methods expect pointers to struct
instance or slices of the same contained with the `interface{}`s**

Now you have your structs prepared to be seralized or materialized, What
about the rest?

### Create Record Example

You can Unmarshal a jsonapi payload using
[jsonapi.UnmarshalPayload](http://godoc.org/github.com/shwoodard/jsonapi#UnmarshalPayload).
It reads from an [io.Reader](https://golang.org/pkg/io/#Reader)
containing a jsonapi payload for one record (but can have related
records).  Then, it materializes a struct that you created and passed in
(using new or &).  Again, the method supports single records only, at
the top level, in request payloads at the moment. Bulk creates and
updates are not supported yet.

After saving your record, you can use,
[MarshalOnePayload](http://godoc.org/github.com/shwoodard/jsonapi#MarshalOnePayload),
to write the jsonapi response to an
[io.Writer](https://golang.org/pkg/io/#Writer).

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

Writes a jsonapi response, with related records sideloaded, into an
`included` array.  This method encodes a response for a single record
only. If you want to serialize many records, see,
[MarshalManyPayload](#marshalmanypayload).

#### Handler Example Code

```go
func CreateBlog(w http.ResponseWriter, r *http.Request) {
	blog := new(Blog)

	if err := jsonapi.UnmarshalPayload(r.Body, blog); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// ...save your blog...

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

Visit [godoc](http://godoc.org/github.com/shwoodard/jsonapi#MarshalManyPayload)

Takes an `io.Writer` and an slice of `interface{}`.  Note, if you have a
type safe array of your structs, like,

```go
var blogs []*Blog
```

you will need to iterate over the slice of `Blog` pointers and append
them to an interface array, like,

```go
blogInterface := make([]interface{}, len(blogs))

for i, blog := range blogs {
  blogInterface[i] = blog
}

```

Alternatively, you can insert your `Blog`s into a slice of `interface{}`
the first time.  For example when you fetch the `Blog`s from the db
`append` them to an `[]interface{}` rather than a `[]*Blog`.  So your
method signature to reach into your data store may look something like
this,

```go
func FetchBlogs() ([]interface{}, error)
```

#### Handler Example Code

```go
func ListBlogs(w http.ResponseWriter, r *http.Request) {
	// ...fetch your blogs, filter, offset, limit, etc...

  // but, for now
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

This method is not strictly meant to for use in implementation code,
although feel free.  It was mainly created for use test; in most cases,
your request payloads for create will be embedded rather than sideloaded
for related records.  This method will serialize a single struct pointer
into an embedded json response.  In other words, there will be no,
`included`, array in the json; all relationships will be serialized
inline with the data.

However, in tests, you may want to construct payloads to post to create
methods that are embedded to most closely model the payloads that will
be produced by the client.  This method aims to enable that.

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

## Alternative Installation
I use git subtrees to manage dependencies rather than `go get` so that
the src is committed to my repo.

```
git subtree add --squash --prefix=src/github.com/shwoodard/jsonapi git@github.com:shwoodard/jsonapi master
```

To update,

```
git subtree pull --squash --prefix=src/github.com/shwoodard/jsonapi git@github.com:shwoodard/jsonapi master
```

This assumes that I have my repo structured with a `src` dir containing
a collection of packages and `GOPATH` is set to the root
folder--containing `src`.

## Contributing

Fork, Change, Pull Request *with tests*.
