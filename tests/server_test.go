package tests

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/seanhagen/list-all-org-prs/server"
	"net/http"
	"reflect"
	"testing"
)

func TestingHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	}
}

func Test_CreateRoute_Good(t *testing.T) {
	out := server.CreateRoute(server.AUTHNONE, TestingHandler())

	if out.AuthType != server.AUTHNONE {
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

func Test_CreateRoute_TokenAuth(t *testing.T) {
	out := server.CreateRoute(server.AUTHTOKEN, TestingHandler())

	if out.AuthType != server.AUTHTOKEN {
		t.Error(fmt.Sprintf("Expected AuthType: %v, got: %v", server.AUTHTOKEN, out.AuthType))
	}
}

func Test_CreateRoute_FixInvalidAuthType(t *testing.T) {
	out := server.CreateRoute(3, TestingHandler())

	if out.AuthType != server.AUTHNONE {
		t.Error(fmt.Sprintf("Expected AuthType: %v, got: %v", server.AUTHNONE, out.AuthType))
	}
}
