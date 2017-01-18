package jsonapi

import "fmt"

const clientIDAnnotation = "client-id"

// OnePayload is used to represent a generic JSON API payload where a single
// resource (Node) was included as an {} in the "data" key
type OnePayload struct {
	Data     *Node        `json:"data"`
	Included []*Node      `json:"included,omitempty"`
	Links    *LinksObject `json:"links,omitempty"`
}

// ManyPayload is used to represent a generic JSON API payload where many
// resources (Nodes) were included in an [] in the "data" key
type ManyPayload struct {
	Data     []*Node      `json:"data"`
	Included []*Node      `json:"included,omitempty"`
	Links    *LinksObject `json:"links,omitempty"`
}

// Node is used to represent a generic JSON API Resource
type Node struct {
	Type          string                 `json:"type"`
	ID            string                 `json:"id"`
	ClientID      string                 `json:"client-id,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
	Links         *LinksObject           `json:"links,omitempty"`
}

// RelationshipOneNode is used to represent a generic has one JSON API relation
type RelationshipOneNode struct {
	Data  *Node        `json:"data"`
	Links *LinksObject `json:"links,omitempty"`
}

// RelationshipManyNode is used to represent a generic has many JSON API
// relation
type RelationshipManyNode struct {
	Data  []*Node      `json:"data"`
	Links *LinksObject `json:"links,omitempty"`
}

// LinksObject is used to represent a `links` object.
// http://jsonapi.org/format/#document-links
type LinksObject map[string]interface{}

func (lo *LinksObject) validate() (err error) {
	// Each member of a links object is a “link”. A link MUST be represented as
	// either:
	//  - a string containing the link’s URL.
	//  - an object (“link object”) which can contain the following members:
	//    - href: a string containing the link’s URL.
	//    - meta: a meta object containing non-standard meta-information about the
	//            link.
	if lo == nil {
		return
	}

	for k, v := range *lo {
		_, isString := v.(string)
		_, isLinkObject := v.(LinkObject)

		if !(isString || isLinkObject) {
			return fmt.Errorf(
				"The %s member of the links object was not a string or link object",
				k,
			)
		}
	}
	return
}

// LinkObject is used to represent a member of the `links` object.
type LinkObject struct {
	Href string                 `json:"href"`
	Meta map[string]interface{} `json:"meta,omitempty"`
}

// Linkable is used to include document links in response data
// e.g. {"self": "http://example.com/posts/1"}
type Linkable interface {
	JSONLinks() *LinksObject
}

// RelationshipLinkable is used to include relationship links  in response data
// e.g. {"related": "http://example.com/posts/1/comments"}
type RelationshipLinkable interface {
	// JSONRelationshipLinks will be invoked for each relationship with the corresponding relation name (e.g. `comments`)
	JSONRelationshipLinks(relation string) *LinksObject
}
