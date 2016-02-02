package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	ErrMissingInput                 = errors.New("error-missing-input")
	ErrUnknownContentType           = errors.New("error-no-content-type")
	ErrUnknownMethod                = errors.New("error-unknown-method")
	ErrIncompatibleType             = errors.New("error-incompatible-type")
	ErrNotSupportedUrlParameterType = errors.New("error-not-supported-url-query-param-type")
	ErrNoHttpHeaderSpec             = errors.New("error-no-http-header-spec")
	ErrNoSignKey                    = errors.New("no-sign-key")
	ErrNoVerifyKey                  = errors.New("no-verify-key")
	ErrInvalidAuthToken             = errors.New("invalid-token")
	ErrExpiredAuthToken             = errors.New("token-expired")
	ErrNoAuthToken                  = errors.New("no-auth-token")

	DefaultErrorRenderer = func(resp http.ResponseWriter, req *http.Request, message string, code int) error {
		resp.WriteHeader(code)
		escaped := message
		if len(message) > 0 {
			escaped = strings.Replace(message, "\"", "'", -1)
		}
		// First look for accept content type in the header
		ct := content_type_for_response(req)
		switch ct {
		case "application/json":
			resp.Write([]byte(fmt.Sprintf("{ \"error\": \"%s\" }", escaped)))
		case "application/protobuf":
		default:
			resp.Write([]byte(fmt.Sprintf("<html><body>Error: %s </body></html>", escaped)))
		}
		return nil
	}
)
