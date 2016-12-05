package jsonapi

const clientIDAnnotation = "client-id"

// OnePayload is used to represent a generic JSON API payload where a single
// resource (Node) was included as an {} in the "data" key
type OnePayload struct {
	Data     *Node              `json:"data"`
	Included []*Node            `json:"included,omitempty"`
	Links    *map[string]string `json:"links,omitempty"`
}

// ManyPayload is used to represent a generic JSON API payload where many
// resources (Nodes) were included in an [] in the "data" key
type ManyPayload struct {
	Data     []*Node            `json:"data"`
	Included []*Node            `json:"included,omitempty"`
	Links    *map[string]string `json:"links,omitempty"`
}

// Node is used to represent a generic JSON API Resource
type Node struct {
	Type          string                 `json:"type"`
	ID            string                 `json:"id"`
	ClientID      string                 `json:"client-id,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
	Links         map[string]interface{} `json:"links,omitempty"`
}

// RelationshipOneNode is used to represent a generic has one JSON API relation
type RelationshipOneNode struct {
	Data  *Node              `json:"data"`
	Links *map[string]string `json:"links,omitempty"`
}

// RelationshipManyNode is used to represent a generic has many JSON API
// relation
type RelationshipManyNode struct {
	Data  []*Node            `json:"data"`
	Links *map[string]string `json:"links,omitempty"`
}
