package data

import "testing"

var newPointerTests = []struct {
	name  string
	pt    PointerType
	valid bool
}{
	{"hello", UserPtr, true},
	{"", UserPtr, true},
	{"hello.world", UserPtr, false},
	{"hello/world", UserPtr, false},
}

func TestNewPointerID(t *testing.T) {
	user := UserID(1)
	for _, tt := range newPointerTests {
		_, err := NewPointerID(user, tt.name, tt.pt)

		if err != nil && tt.valid == true || err == nil && tt.valid == false {
			t.Errorf("NewPointerID(%d, %s, %s) => %s, valid should be %t", user, tt.name, tt.pt, err, tt.valid)
		}
	}
}
