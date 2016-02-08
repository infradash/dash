package encoding

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"io"
)

type nht int

const (
	no_header nht = 1
)
const (
	ContentTypeDefault  ContentType = ContentType("")
	ContentTypeAny      ContentType = ContentType("*/*")
	ContentTypeJSON     ContentType = ContentType("application/json")
	ContentTypeProtobuf ContentType = ContentType("application/protobuf")
	ContentTypeHTML     ContentType = ContentType("text/html")
	ContentTypePlain    ContentType = ContentType("text/plain")
)

type ContentType string

func (this ContentType) String() string {
	return string(this)
}

func MarshalString(w io.Writer, value interface{}) error {
	switch value := value.(type) {
	case *string:
		w.Write([]byte(*value))
	case string:
		w.Write([]byte(value))
	case []byte:
		w.Write(value)
	default:
		fmt.Fprintf(w, "%v", value)
	}
	return nil
}

func MarshalJSON(resp io.Writer, typed interface{}) error {
	if buff, err := json.Marshal(typed); err == nil {
		resp.Write(buff)
		return nil
	} else {
		return err
	}
}

func MarshalProtobuf(resp io.Writer, any interface{}) error {
	typed, ok := any.(proto.Message)
	if !ok {
		return ErrIncompatibleType
	}
	if buff, err := proto.Marshal(typed); err == nil {
		resp.Write(buff)
		return nil
	} else {
		return err
	}
}

var (
	marshalers = map[ContentType]func(io.Writer, interface{}) error{
		ContentTypeDefault:  MarshalJSON,
		ContentTypeAny:      MarshalJSON,
		ContentTypeJSON:     MarshalJSON,
		ContentTypeProtobuf: MarshalProtobuf,
		ContentTypePlain:    MarshalString,
		ContentTypeHTML:     nil,
	}
)

// Returns the ContentType given the string.
func ContentTypeFromString(t string) (ContentType, error) {
	if Check(ContentType(t)) {
		return ContentType(t), nil
	}
	return ContentTypeDefault, ErrBadContentType
}

func Check(c ContentType) bool {
	_, canMarshal := marshalers[c]
	_, canUnmarshal := unmarshalers[c]
	return canMarshal && canUnmarshal
}

func Marshal(t ContentType, writer io.Writer, value interface{}) error {
	if m, has := marshalers[t]; has {
		return m(writer, value)
	}
	return ErrUnknownContentType
}
