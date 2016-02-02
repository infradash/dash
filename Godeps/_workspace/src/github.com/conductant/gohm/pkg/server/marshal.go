package server

import (
	"encoding/json"
	"errors"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"io"
	"io/ioutil"
	"net/http"
)

var (
	string_marshaler = func(contentType string, resp http.ResponseWriter, typed interface{}, noHeader ...nht) error {
		if str, ok := typed.(*string); ok {
			if len(noHeader) == 0 {
				resp.Header().Add("Content-Type", contentType)
			}
			resp.Write([]byte(*str))
			return nil
		} else {
			return errors.New("wrong-type-expects-str-ptr")
		}
	}

	string_unmarshaler = func(body io.ReadCloser, typed interface{}) error {
		if _, ok := typed.(*string); !ok {
			return errors.New("wrong-type-expects-str-ptr")
		}
		if buff, err := ioutil.ReadAll(body); err == nil {
			ptr := typed.(*string)
			*ptr = string(buff)
			return nil
		} else {
			return err
		}
	}

	json_marshaler = func(contentType string, resp http.ResponseWriter, typed interface{}, noHeader ...nht) error {
		if buff, err := json.Marshal(typed); err == nil {
			if len(noHeader) == 0 {
				resp.Header().Add("Content-Type", contentType)
			}
			resp.Write(buff)
			return nil
		} else {
			return err
		}
	}

	json_unmarshaler = func(body io.ReadCloser, typed interface{}) error {
		dec := json.NewDecoder(body)
		return dec.Decode(typed)
	}

	json_logging_unmarshaler = func(body io.ReadCloser, typed interface{}) error {
		buff, err := ioutil.ReadAll(body)
		if err != nil {
			return err
		}
		glog.V(100).Infoln("Unmarshal [", string(buff), "]")
		return json.Unmarshal(buff, typed)
	}

	proto_marshaler = func(contentType string, resp http.ResponseWriter, any interface{}, noHeader ...nht) error {
		typed, ok := any.(proto.Message)
		if !ok {
			return ErrIncompatibleType
		}
		if buff, err := proto.Marshal(typed); err == nil {
			resp.Header().Add("Content-Type", contentType)
			resp.Write(buff)
			return nil
		} else {
			return err
		}
	}

	proto_unmarshaler = func(body io.ReadCloser, any interface{}) error {
		typed, ok := any.(proto.Message)
		if !ok {
			return ErrIncompatibleType
		}
		buff, err := ioutil.ReadAll(body)
		if err != nil {
			return err
		}
		return proto.Unmarshal(buff, typed)
	}

	marshalers = map[string]func(string, http.ResponseWriter, interface{}, ...nht) error{
		string(ContentTypeDefault):  json_marshaler,
		string(ContentTypeJSON):     json_marshaler,
		string(ContentTypeProtobuf): proto_marshaler,
		string(ContentTypePlain):    string_marshaler,
		string(ContentTypeHTML):     nil,
	}

	unmarshalers = map[string]func(io.ReadCloser, interface{}) error{
		string(ContentTypeDefault):  json_logging_unmarshaler,
		string(ContentTypeJSON):     json_logging_unmarshaler,
		string(ContentTypeProtobuf): proto_unmarshaler,
		string(ContentTypePlain):    string_unmarshaler,
		string(ContentTypeHTML):     nil,
	}
)

func UnmarshalProtobuff(req *http.Request, typed proto.Message) (err error) {
	contentType := content_type_for_request(req)
	if unmarshaler, has := unmarshalers[contentType]; has {
		return unmarshaler(req.Body, typed)
	} else {
		return ErrUnknownContentType
	}
}

func MarshalProtobuff(req *http.Request, typed proto.Message, resp http.ResponseWriter) (err error) {
	contentType := content_type_for_response(req)
	if marshaler, has := marshalers[contentType]; has {
		return marshaler(contentType, resp, typed)
	} else {
		return ErrUnknownContentType
	}
}

func Unmarshal(req *http.Request, any interface{}) (err error) {
	contentType := content_type_for_request(req)
	if unmarshaler, has := unmarshalers[contentType]; has {
		return unmarshaler(req.Body, any)
	} else {
		return ErrUnknownContentType
	}
}

func Marshal(req *http.Request, any interface{}, resp http.ResponseWriter) (err error) {
	if buff, err := json.Marshal(any); err == nil {
		resp.Header().Add("Content-Type", "application/json")
		resp.Write(buff)
		return nil
	} else {
		return err
	}
}

func content_type_for_request(req *http.Request) string {
	t := "application/json"

	if req.Method == "POST" || req.Method == "PUT" {
		t = req.Header.Get("Content-Type")
	}
	switch t {
	case "*/*":
		return "application/json"
	case "":
		return "application/json"
	default:
		return t
	}
}

func content_type_for_response(req *http.Request) string {
	t := req.Header.Get("Accept")
	switch t {
	case "*/*":
		return "application/json"
	case "":
		return content_type_for_request(req) // use the same content type as the request if no accept header
	default:
		return t
	}
}
