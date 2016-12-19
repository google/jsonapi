package jsonapi

const clientIDAnnotation = "client-id"

// OnePayload is used to represent a generic JSON API payload where a single
// resource (Node) was included as an {} in the "data" key
type OnePayload struct {
	Data     *Node            `json:"data"`
	Included []*Node          `json:"included,omitempty"`
	Links    *map[string]Link `json:"links,omitempty"`
}

// ManyPayload is used to represent a generic JSON API payload where many
// resources (Nodes) were included in an [] in the "data" key
type ManyPayload struct {
	Data     []*Node          `json:"data"`
	Included []*Node          `json:"included,omitempty"`
	Links    *map[string]Link `json:"links,omitempty"`
}

// Node is used to represent a generic JSON API Resource
type Node struct {
	Type          string                 `json:"type"`
	ID            string                 `json:"id"`
	ClientID      string                 `json:"client-id,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
	Links         *map[string]Link       `json:"links,omitempty"`
}

// RelationshipOneNode is used to represent a generic has one JSON API relation
type RelationshipOneNode struct {
	Data  *Node            `json:"data"`
	Links *map[string]Link `json:"links,omitempty"`
}

// RelationshipManyNode is used to represent a generic has many JSON API
// relation
type RelationshipManyNode struct {
	Data  []*Node          `json:"data"`
	Links *map[string]Link `json:"links,omitempty"`
}

// Link is used to represent a `links` object.
// http://jsonapi.org/format/#document-links
type Link struct {
	Href string                 `json:"href"`
	Meta map[string]interface{} `json:"meta,omitempty"`
}

// Linkable is used to include document links in response data
// e.g. {"self": "http://example.com/posts/1"}
type Linkable interface {
	JSONLinks() *map[string]Link
}

// RelationshipLinkable is used to include relationship links  in response data
// e.g. {"related": "http://example.com/posts/1/comments"}
type RelationshipLinkable interface {
	// JSONRelationshipLinks will be invoked for each relationship with the corresponding relation name (e.g. `comments`)
	JSONRelationshipLinks(relation string) *map[string]Link
}
