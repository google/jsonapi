package jsonapi

type JsonApiNodeWrapper struct {
	Data *JsonApiNode `json:"data"`
}

type JsonApiNode struct {
	Type          string                 `json:"type"`
	Id            string                 `json:"id"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Relationships map[string]interface{} `json:"realtionships,omitempty"`
}
