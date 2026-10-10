package cli

import (
	"reflect"
	"testing"
)

func TestParseSetFlags(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  map[string]any
	}{
		{"single value", []string{"a=1"}, map[string]any{"a": "1"}},
		{"comma separated", []string{"a=1,b=2"}, map[string]any{"a": "1", "b": "2"}},
		{"repeatable", []string{"a=1", "b=2"}, map[string]any{"a": "1", "b": "2"}},
		{"list", []string{"a={x,y,z}"}, map[string]any{"a": []string{"x", "y", "z"}}},
		{"empty list", []string{"a={}"}, map[string]any{"a": []string{}}},
		{"mixed list and single value", []string{"a={x,y},b=1"}, map[string]any{"a": []string{"x", "y"}, "b": "1"}},
		{"value containing equals sign", []string{"a=b=c"}, map[string]any{"a": "b=c"}},
		{"escaped comma", []string{`a=x\,y,b=2`}, map[string]any{"a": "x,y", "b": "2"}},
		{"escaped equals sign", []string{`a\=b=c`}, map[string]any{"a=b": "c"}},
		{"commas inside a list do not split", []string{"a={x,y},b={z}"}, map[string]any{"a": []string{"x", "y"}, "b": []string{"z"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseSetFlags(c.input)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestParseSetFlagsError(t *testing.T) {
	if _, err := parseSetFlags([]string{"noequal"}); err == nil {
		t.Error("missing = should error")
	}
}
