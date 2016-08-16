package server

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"reflect"
	"testing"
)

func TestingHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	}
}

func Test_createRoute_Good(t *testing.T) {
	out := createRoute(AUTHNONE, TestingHandler())

	if out.AuthType != AUTHNONE {
		t.Error("AuthType does not match")
	}

	if out.Handler == nil {
		t.Error("Handler is nil")
	}

	x := reflect.TypeOf(out.Handler).Kind()

	if x != 0x13 {
		t.Errorf("Handler is not right type, expected %#v, got: %#v", 0x13, x)
	}
}

func Test_createRoute_TokenAuth(t *testing.T) {
	out := createRoute(AUTHTOKEN, TestingHandler())

	if out.AuthType != AUTHTOKEN {
		t.Error(fmt.Sprintf("Expected AuthType: %v, got: %v", AUTHTOKEN, out.AuthType))
	}
}

func Test_createRoute_FixInvalidAuthType(t *testing.T) {
	out := createRoute(3, TestingHandler())

	if out.AuthType != AUTHNONE {
		t.Error(fmt.Sprintf("Expected AuthType: %v, got: %v", AUTHNONE, out.AuthType))
	}
}
