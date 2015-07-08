package jsonapi

type JsonApiOnePayload struct {
	Data     *JsonApiNode       `json:"data"`
	Included []*JsonApiNode     `json:"included,omitempty"`
	Links    *map[string]string `json:"links,omitempty"`
}

type JsonApiManyPayload struct {
	Data     []*JsonApiNode     `json:"data"`
	Included []*JsonApiNode     `json:"included,omitempty"`
	Links    *map[string]string `json:"links,omitempty"`
}

type Models interface {
	GetData() []interface{}
}

type JsonApiNode struct {
	Type          string                 `json:"type"`
	Id            string                 `json:"id"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
}

type JsonApiRelationshipOneNode struct {
	Data  *JsonApiNode       `json:"data"`
	Links *map[string]string `json:"links,omitempty"`
}

type JsonApiRelationshipManyNode struct {
	Data  []*JsonApiNode     `json:"data"`
	Links *map[string]string `json:"links,omitempty"`
}
