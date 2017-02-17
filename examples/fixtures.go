package main

import "time"

func fixtureBlogCreate(i int) *Blog {
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

func fixtureBlogsList() (blogs []interface{}) {
	for i := 0; i < 10; i++ {
		blogs = append(blogs, fixtureBlogCreate(i))
	}

	return blogs
}
