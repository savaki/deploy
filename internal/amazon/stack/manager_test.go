package stack

import (
	"io/ioutil"
	"testing"
)

func Test_getParameters(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		data, err := ioutil.ReadFile("testdata/a/table.template")
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}

		parameters, err := getParameters(string(data), nil)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		if got, want := len(parameters), 0; got != want {
			t.Fatalf("got %v; want %v", got, want)
		}
	})

	t.Run("some", func(t *testing.T) {
		data, err := ioutil.ReadFile("testdata/parameters-some.template")
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}

		all := map[string]string{
			"Foo": "Foo",
			"Bar": "Bar",
		}

		parameters, err := getParameters(string(data), all)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		if got, want := len(parameters), 2; got != want {
			t.Fatalf("got %v; want %v", got, want)
		}
	})
}
