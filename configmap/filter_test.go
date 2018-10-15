package configmap

import "testing"

type foo struct{}
type bar struct{}

func TestTypeFilter(t *testing.T) {
	count := 0

	var f = func(name string, value interface{}) {
		count++
	}

	f("foo", &foo{})
	f("bar", &bar{})

	if want, got := 2, count; want != got {
		t.Fatalf("plain call: count: want %v, got %v", want, got)
	}

	filtered := TypeFilter(&foo{})(f)

	filtered("foo", &foo{})
	filtered("bar", &bar{})

	if want, got := 3, count; want != got {
		t.Fatalf("filtered call: count: want %v, got %v", want, got)
	}
}
