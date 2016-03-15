package encoding

import (
	"encoding/json"
	"errors"
	"github.com/golang/protobuf/proto"
	"io"
	"io/ioutil"
)

func UnmarshalString(body io.ReadCloser, typed interface{}) error {
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

func UnmarshalJSON(body io.ReadCloser, typed interface{}) error {
	dec := json.NewDecoder(body)
	return dec.Decode(typed)
}

func UnmarshalProtobuf(body io.ReadCloser, any interface{}) error {
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

var (
	unmarshalers = map[ContentType]func(io.ReadCloser, interface{}) error{
		ContentTypeDefault:  UnmarshalJSON,
		ContentTypeJSON:     UnmarshalJSON,
		ContentTypeProtobuf: UnmarshalProtobuf,
		ContentTypePlain:    UnmarshalString,
		ContentTypeHTML:     nil,
	}
)

func Unmarshal(t ContentType, reader io.ReadCloser, value interface{}) error {
	if u, has := unmarshalers[t]; has {
		return u(reader, value)
	}
	return ErrUnknownContentType
}
