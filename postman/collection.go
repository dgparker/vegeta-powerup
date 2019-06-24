package postman

import (
	"net/http"
	"strings"
)

// Collection represents a postman collection
type Collection struct {
	Info  CollectionInfo   `json:"info"`
	Items []CollectionItem `json:"item"`
}

// CollectionInfo ...
type CollectionInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CollectionItem ...
type CollectionItem struct {
	Name      string                  `json:"name"`
	Request   CollectionItemRequest   `json:"request"`
	Responses CollectionItemResponses `json:"response"`
}

// CollectionItemRequest ...
type CollectionItemRequest struct {
	URL         CollectionItemRequestURL `json:"url"`
	Method      string                   `json:"method"`
	Header      []RequestHeader          `json:"header"`
	Body        RequestBody              `json:"body"`
	Description string                   `json:"description"`
}

// WrapHeaders transforms the postman header values to a go http.Header
func (r *CollectionItemRequest) WrapHeaders() http.Header {
	if len(r.Header) == 0 {
		return http.Header{}
	}

	hdr := http.Header{}
	for _, v := range r.Header {
		hdr.Add(v.Key, v.Value)
	}
	return hdr
}

// CollectionItemRequestURL ...
type CollectionItemRequestURL struct {
	Raw   string   `json:"raw"`
	Host  []string `json:"host"`
	Path  []string `json:"path"`
	Query []Query  `json:"query"`
}

// Query ...
type Query struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CollectionItemResponse ...
type CollectionItemResponse struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Status string           `json:"status"`
	Code   int              `json:"code"`
	Header []ResponseHeader `json:"header"`
	Body   string           `json:"body"`
}

// CollectionItemResponses ...
type CollectionItemResponses []CollectionItemResponse

// RequestHeader ...
type RequestHeader struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	Disabled    bool   `json:"disabled"`
}

// RequestBody ...
type RequestBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}

// Bytes returns the byte representation of the json string
// if empty returns []byte{}
func (r RequestBody) Bytes() []byte {
	if r.Raw == "" {
		return []byte{}
	}

	// remove postman formatting
	r.Raw = strings.ReplaceAll(r.Raw, "\t", "")
	r.Raw = strings.ReplaceAll(r.Raw, "\n", "")
	return []byte(r.Raw)
}

// ResponseHeader ...
type ResponseHeader struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
