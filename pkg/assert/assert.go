package assert

import (
	"reflect"
	"testing"
)

// Equal checks if values are equal
func Equal(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		return
	}
	t.Errorf("Received %v (type %v), expected %v (type %v)",
		a, reflect.TypeOf(a), b, reflect.TypeOf(b))
}

func True(t *testing.T, value bool, msgAndArgs ...interface{}) bool {
	if value {
		return true
	}
	if len(msgAndArgs) > 0 {
		t.Errorf("Should be true: "+msgAndArgs[0].(string), msgAndArgs[1:]...)
	} else {
		t.Error("Should be true")
	}
	return false
}
