package jsonapi

type OnePayload struct {
	Data     *Node              `json:"data"`
	Included []*Node            `json:"included,omitempty"`
	Links    *map[string]string `json:"links,omitempty"`
}

type ManyPayload struct {
	Data     []*Node            `json:"data"`
	Included []*Node            `json:"included,omitempty"`
	Links    *map[string]string `json:"links,omitempty"`
}

type Node struct {
	Type          string                 `json:"type"`
	Id            string                 `json:"id"`
	ClientId      string                 `json:"client-id,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
}

type RelationshipOneNode struct {
	Data  *Node              `json:"data"`
	Links *map[string]string `json:"links,omitempty"`
}

type RelationshipManyNode struct {
	Data  []*Node            `json:"data"`
	Links *map[string]string `json:"links,omitempty"`
}
