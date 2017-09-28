package main

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/jsonapi"
)

// Blog is a model representing a blog site
type Blog struct {
	ID            int       `jsonapi:"primary,blogs"`
	Title         string    `jsonapi:"attr,title"`
	Posts         []*Post   `jsonapi:"relation,posts"`
	CurrentPost   *Post     `jsonapi:"relation,current_post"`
	CurrentPostID int       `jsonapi:"attr,current_post_id"`
	CreatedAt     time.Time `jsonapi:"attr,created_at"`
	ViewCount     int       `jsonapi:"attr,view_count"`
}

// Post is a model representing a post on a blog
type Post struct {
	ID       int        `jsonapi:"primary,posts"`
	BlogID   int        `jsonapi:"attr,blog_id"`
	Title    string     `jsonapi:"attr,title"`
	Body     string     `jsonapi:"attr,body"`
	Comments []*Comment `jsonapi:"relation,comments"`
}

// Comment is a model representing a user submitted comment
type Comment struct {
	ID     int    `jsonapi:"primary,comments"`
	PostID int    `jsonapi:"attr,post_id"`
	Body   string `jsonapi:"attr,body"`
}

// JSONAPILinks implements the Linkable interface for a blog
func (blog Blog) JSONAPILinks(ctx context.Context) *jsonapi.Links {
	baseURI := baseURI(ctx)
	return &jsonapi.Links{
		"self": fmt.Sprintf("%s/blogs/%d", baseURI, blog.ID),
	}
}

// JSONAPIRelationshipLinks implements the RelationshipLinkable interface for a blog
func (blog Blog) JSONAPIRelationshipLinks(ctx context.Context, relation string) *jsonapi.Links {
	baseURI := baseURI(ctx)
	if relation == "posts" {
		return &jsonapi.Links{
			"related": fmt.Sprintf("%s/blogs/%d/posts", baseURI, blog.ID),
		}
	}
	if relation == "current_post" {
		return &jsonapi.Links{
			"related": fmt.Sprintf("%s/blogs/%d/current_post", baseURI, blog.ID),
		}
	}
	return nil
}

// JSONAPIMeta implements the Metable interface for a blog
func (blog Blog) JSONAPIMeta(ctx context.Context) *jsonapi.Meta {
	return &jsonapi.Meta{
		"detail": "extra details regarding the blog",
	}
}

// JSONAPIRelationshipMeta implements the RelationshipMetable interface for a blog
func (blog Blog) JSONAPIRelationshipMeta(ctx context.Context, relation string) *jsonapi.Meta {
	if relation == "posts" {
		return &jsonapi.Meta{
			"detail": "posts meta information",
		}
	}
	if relation == "current_post" {
		return &jsonapi.Meta{
			"detail": "current post meta information",
		}
	}
	return nil
}

func baseURI(ctx context.Context) string {
	requestURI, ok := ctx.Value(keyRequestURI).(string)
	if !ok {
		return ""
	}
	u, _ := url.Parse(requestURI)
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
}
