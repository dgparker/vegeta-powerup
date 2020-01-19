package postman

import (
	"net/http"
)

type Collection struct {
	Info                    Info                    `json:"info"`
	Items                   []Item                  `json:"item"`
	ProtocolProfileBehavior ProtocolProfileBehavior `json:"protocolProfileBehavior"`
}

type Info struct {
	PostmanID string `json:"_postman_id"`
	Name      string `json:"name"`
	Schema    string `json:"schema"`
}

type AuthItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type Auth struct {
	Type   string     `json:"type"`
	APIKey []AuthItem `json:"apikey"`
	Bearer []AuthItem `json:"bearer"`
	Basic  []AuthItem `json:"basic"`
}

type Header struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Type     string `json:"type"`
	Name     string `json:"name,omitempty"`
	Disabled bool   `json:"disabled"`
}

type Raw struct {
	Language string `json:"language"`
}

type Options struct {
	Raw Raw `json:"raw"`
}

type Body struct {
	Mode    string  `json:"mode"`
	Raw     string  `json:"raw"`
	Options Options `json:"options"`
}

type Query struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type URL struct {
	Raw   string   `json:"raw"`
	Host  []string `json:"host"`
	Path  []string `json:"path"`
	Query []Query  `json:"query"`
}

type Request struct {
	Auth   Auth     `json:"auth"`
	Method string   `json:"method"`
	Header []Header `json:"header"`
	Body   Body     `json:"body"`
	URL    URL      `json:"url"`
}

type Item struct {
	Name     string        `json:"name"`
	Request  Request       `json:"request"`
	Response []interface{} `json:"response"`
	Items    []Item        `json:"item"`
}
type ProtocolProfileBehavior struct {
}

// WrapHeaders transforms the postman header values to a go http.Header
func (r *Request) WrapHeaders() http.Header {
	if len(r.Header) == 0 {
		return http.Header{}
	}

	hdr := http.Header{}
	for _, v := range r.Header {
		if !v.Disabled {
			hdr.Add(v.Key, v.Value)
		}
	}
	return hdr
}

// Bytes returns the byte representation of the json string
// if empty returns []byte{}
func (r Body) Bytes() []byte {
	if r.Raw == "" {
		return []byte{}
	}

	return []byte(r.Raw)
}
