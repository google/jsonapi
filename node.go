package jsonapi

import "fmt"

type nodeError bool

const dominantFieldConflict nodeError = false

// Payloader is used to encapsulate the One and Many payload types
type Payloader interface {
	clearIncluded()
}

// OnePayload is used to represent a generic JSON API payload where a single
// resource (Node) was included as an {} in the "data" key
type OnePayload struct {
	Data     *Node   `json:"data"`
	Included []*Node `json:"included,omitempty"`
	Links    *Links  `json:"links,omitempty"`
	Meta     *Meta   `json:"meta,omitempty"`
}

func (p *OnePayload) clearIncluded() {
	p.Included = []*Node{}
}

// ManyPayload is used to represent a generic JSON API payload where many
// resources (Nodes) were included in an [] in the "data" key
type ManyPayload struct {
	Data     []*Node `json:"data"`
	Included []*Node `json:"included,omitempty"`
	Links    *Links  `json:"links,omitempty"`
	Meta     *Meta   `json:"meta,omitempty"`
}

func (p *ManyPayload) clearIncluded() {
	p.Included = []*Node{}
}

type attributes map[string]interface{}

// Node is used to represent a generic JSON API Resource
type Node struct {
	Type          string                 `json:"type"`
	ID            string                 `json:"id,omitempty"`
	ClientID      string                 `json:"client-id,omitempty"`
	Attributes    attributes             `json:"attributes,omitempty"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
	Links         *Links                 `json:"links,omitempty"`
	Meta          *Meta                  `json:"meta,omitempty"`
}

func isNodeError(i interface{}) bool {
	_, ok := i.(nodeError)
	return ok
}

func (n *Node) cleanupDominantFieldIssues() {
	for k, v := range n.Attributes {
		if isNodeError(v) {
			delete(n.Attributes, k)
		}
	}
}

func (a attributes) set(k string, v interface{}) {
	if _, ok := a[k]; ok {
		a[k] = dominantFieldConflict
	} else {
		a[k] = v
	}
}

func (n *Node) mergeAttributes(attrs attributes) {
	for k, v := range attrs {
		n.Attributes[k] = v
	}
}

func combineNodes(nodes []*Node) *Node {
	n := &Node{}
	for _, node := range nodes {
		n.peerMerge(node)
	}
	return n
}

func (n *Node) peerMerge(node *Node) {
	n.mergeFunc(node, func(attrs attributes) {
		for k, v := range node.Attributes {
			n.Attributes.set(k, v)
		}
	})
}

func (n *Node) merge(node *Node) {
	n.mergeFunc(node, n.mergeAttributes)
}

func (n *Node) mergeFunc(node *Node, attrSetter func(attrs attributes)) {
	if node.Type != "" {
		n.Type = node.Type
	}

	if node.ID != "" {
		n.ID = node.ID
	}

	if node.ClientID != "" {
		n.ClientID = node.ClientID
	}

	if n.Attributes == nil && node.Attributes != nil {
		n.Attributes = make(map[string]interface{})
	}
	attrSetter(node.Attributes)

	if n.Relationships == nil && node.Relationships != nil {
		n.Relationships = make(map[string]interface{})
	}
	for k, v := range node.Relationships {
		n.Relationships[k] = v
	}

	if node.Links != nil {
		n.Links = node.Links
	}
}

// RelationshipOneNode is used to represent a generic has one JSON API relation
type RelationshipOneNode struct {
	Data  *Node  `json:"data"`
	Links *Links `json:"links,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
}

// RelationshipManyNode is used to represent a generic has many JSON API
// relation
type RelationshipManyNode struct {
	Data  []*Node `json:"data"`
	Links *Links  `json:"links,omitempty"`
	Meta  *Meta   `json:"meta,omitempty"`
}

// Links is used to represent a `links` object.
// http://jsonapi.org/format/#document-links
type Links map[string]interface{}

func (l *Links) validate() (err error) {
	// Each member of a links object is a “link”. A link MUST be represented as
	// either:
	//  - a string containing the link’s URL.
	//  - an object (“link object”) which can contain the following members:
	//    - href: a string containing the link’s URL.
	//    - meta: a meta object containing non-standard meta-information about the
	//            link.
	for k, v := range *l {
		_, isString := v.(string)
		_, isLink := v.(Link)

		if !(isString || isLink) {
			return fmt.Errorf(
				"The %s member of the links object was not a string or link object",
				k,
			)
		}
	}
	return
}

// Link is used to represent a member of the `links` object.
type Link struct {
	Href string `json:"href"`
	Meta Meta   `json:"meta,omitempty"`
}

// Linkable is used to include document links in response data
// e.g. {"self": "http://example.com/posts/1"}
type Linkable interface {
	JSONAPILinks() *Links
}

// RelationshipLinkable is used to include relationship links  in response data
// e.g. {"related": "http://example.com/posts/1/comments"}
type RelationshipLinkable interface {
	// JSONAPIRelationshipLinks will be invoked for each relationship with the corresponding relation name (e.g. `comments`)
	JSONAPIRelationshipLinks(relation string) *Links
}

// Meta is used to represent a `meta` object.
// http://jsonapi.org/format/#document-meta
type Meta map[string]interface{}

// Metable is used to include document meta in response data
// e.g. {"foo": "bar"}
type Metable interface {
	JSONAPIMeta() *Meta
}

// RelationshipMetable is used to include relationship meta in response data
type RelationshipMetable interface {
	// JSONRelationshipMeta will be invoked for each relationship with the corresponding relation name (e.g. `comments`)
	JSONAPIRelationshipMeta(relation string) *Meta
}

// derefs the arg, and clones the map-type attributes
// note: maps are reference types, so they need an explicit copy.
func deepCopyNode(n *Node) *Node {
	if n == nil {
		return n
	}

	copyMap := func(m map[string]interface{}) map[string]interface{} {
		if m == nil {
			return m
		}
		cp := make(map[string]interface{})
		for k, v := range m {
			cp[k] = v
		}
		return cp
	}

	copy := *n
	copy.Attributes = copyMap(copy.Attributes)
	copy.Relationships = copyMap(copy.Relationships)
	if copy.Links != nil {
		tmp := Links(copyMap(map[string]interface{}(*copy.Links)))
		copy.Links = &tmp
	}
	if copy.Meta != nil {
		tmp := Meta(copyMap(map[string]interface{}(*copy.Meta)))
		copy.Meta = &tmp
	}
	return &copy
}
