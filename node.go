package jsonapi

type JsonApiPayload struct {
	Data     *JsonApiNode       `json:"data"`
	Included []*JsonApiNode     `json:"included,omitempty"`
	Links    *map[string]string `json:"links,omitempty"`
}

type JsonApiNode struct {
	Type          string                 `json:"type"`
	Id            string                 `json:"id"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Relationships map[string]interface{} `json:"realtionships,omitempty"`
}

type JsonApiRelationshipSingleNode struct {
	Data  *JsonApiNode       `json:"data"`
	Links *map[string]string `json:"links,omitempty"`
}

type JsonApiRelationshipMultipleNode struct {
	Data  []*JsonApiNode     `json:"data"`
	Links *map[string]string `json:"links,omitempty"`
}
