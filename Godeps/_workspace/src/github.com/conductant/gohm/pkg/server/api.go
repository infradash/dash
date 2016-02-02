package server

import (
	"net/http"
	"reflect"
)

type AuthScope string
type EventKey string

type HttpMethod string
type ContentType string
type QueryDefault interface{}
type UrlQueries map[string]QueryDefault
type FormParams UrlQueries
type HttpHeaders map[string]string

var (
	NotDefined = ServiceMethod{}

	AuthScopeNone = AuthScope("*")

	HEAD      HttpMethod = HttpMethod("HEAD")
	PATCH     HttpMethod = HttpMethod("PATCH")
	GET       HttpMethod = HttpMethod("GET")
	POST      HttpMethod = HttpMethod("POST")
	PUT       HttpMethod = HttpMethod("PUT")
	DELETE    HttpMethod = HttpMethod("DELETE")
	MULTIPART HttpMethod = HttpMethod("POST")

	ContentTypeDefault  ContentType = ContentType("")
	ContentTypeJSON     ContentType = ContentType("application/json")
	ContentTypeProtobuf ContentType = ContentType("application/protobuf")
	ContentTypeHTML     ContentType = ContentType("text/html")
	ContentTypePlain    ContentType = ContentType("text/plain")
)

type ServiceMethod struct {
	Doc                  string                          `json:"doc,omitempty"`
	UrlRoute             string                          `json:"route,omitempty"`
	HttpHeaders          HttpHeaders                     `json:"headers,omitempty"`
	HttpMethod           HttpMethod                      `json:"method,omitempty"`
	HttpMethods          []HttpMethod                    `json:"methods,omitempty"`
	UrlQueries           UrlQueries                      `json:"queries,omitempty"`
	FormParams           FormParams                      `json:"params,omitempty"`
	ContentType          ContentType                     `json:"contentType,omitempty"`
	RequestBody          func(*http.Request) interface{} `json:"requestBody,omitempty"`
	ResponseBody         func(*http.Request) interface{} `json:"responseBody,omitempty"`
	CallbackEvent        EventKey                        `json:"callbackEvent,omitempty"`
	CallbackBodyTemplate string                          `json:"callbackBody,omitempty"`
	AuthScope            AuthScope                       `json:"scope,omitempty"`
}

func (sm ServiceMethod) Equals(other ServiceMethod) bool {
	return reflect.DeepEqual(sm, other)
}
