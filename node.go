package fastjsonapi

//easyjson:json
type OnePayload struct {
	Data     *Node              `json:"data"`
	Included []*Node            `json:"included,omitempty"`
	Links    *map[string]string `json:"links,omitempty"`
}

//easyjson:json
type ManyPayload struct {
	Data     []*Node            `json:"data"`
	Included []*Node            `json:"included,omitempty"`
	Links    *map[string]string `json:"links,omitempty"`
}

//easyjson:json
type Node struct {
	Type          string                 `json:"type"`
	ID            string                 `json:"id"`
	ClientID      string                 `json:"client-id,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
}

//easyjson:json
type RelationshipOneNode struct {
	Data  *Node              `json:"data"`
	Links *map[string]string `json:"links,omitempty"`
}

//easyjson:json
type RelationshipManyNode struct {
	Data  []*Node            `json:"data"`
	Links *map[string]string `json:"links,omitempty"`
}
