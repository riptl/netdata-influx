package netdata

import (
	"net/http"
	"net/url"
	"path"
	"strconv"
)

type RequestBuilder struct {
	// Base API URL
	BaseURL string
	Chart string
	// Number of points
	Points int
}

func (b *RequestBuilder) Build() (*http.Request, error) {
	u, err := url.Parse(b.BaseURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, "v1/data")
	v := make(url.Values)
	v.Set("chart", b.Chart)
	v.Set("format", "json")
	v.Set("options", "absolute|jsonwrap")
	if b.Points != 0 {
		v.Set("points", strconv.Itoa(b.Points))
	}
	u.RawQuery = v.Encode()
	return http.NewRequest("GET", u.String(), nil)
}
