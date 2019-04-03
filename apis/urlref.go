package apis

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// URLRef is a wrapper to url.URL.
// It has custom json marshal methods that enable it to be used in K8s CRDs
// such that the CRD resource will have the URL but operator code can can work with url.URL struct
type URLRef struct {
	url.URL
}

// ParseURLRef attempts to parse the given string as a URI-Reference.
func ParseURLRef(u string) *URLRef {
	if u == "" {
		return nil
	}
	pu, err := url.Parse(u)
	if err != nil {
		return nil
	}
	return &URLRef{URL: *pu}
}

// MarshalJSON implements a custom json marshal method used when this type is
// marshaled using json.Marshal.
// json.Marshaler impl
func (u URLRef) MarshalJSON() ([]byte, error) {
	b := fmt.Sprintf("%q", u.String())
	return []byte(b), nil
}

// UnmarshalJSON implements the json unmarshal method used when this type is
// unmarsheled using json.Unmarshal.
// json.Unmarshaler impl
func (u *URLRef) UnmarshalJSON(b []byte) error {
	var ref string
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	r := ParseURLRef(ref)
	if r != nil {
		*u = *r
	}
	return nil
}

// String returns the full string representation of the URI-Reference.
func (u *URLRef) String() string {
	if u == nil {
		return ""
	}
	return u.URL.String()
}
