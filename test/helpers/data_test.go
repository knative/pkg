package helpers

import (
	"fmt"
	"regexp"
)

var matcher = regexp.MustCompile("abcd-[a-z]{8}")

func ExampleAppendRandomString() {
	const s = "abcd"
	t := AppendRandomString(s)
	o := AppendRandomString(s)
	fmt.Println(matcher.MatchString(t), matcher.MatchString(o), o != t)
	// Output: true true true
}
