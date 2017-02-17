package main

import (
	"fmt"
	"time"
)

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
