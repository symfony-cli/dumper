package dumper

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

func init() {
	RegisterCustomDumper(http.Response{}, dumpHttpResponse)
	RegisterCustomDumper(http.Request{}, dumpHttpRequest)
}

func dumpHttpHeaders(s State, headers http.Header) {
	s.Pad()
	s.Write([]byte("Headers: {\n"))
	s.DepthDown()
	keys := make([]string, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		for _, v := range headers[key] {
			s.Pad()
			s.DumpString(key)
			s.Write([]byte(": "))
			s.Dump(v)
			s.Write([]byte(",\n"))
		}
	}
	s.DepthUp()
	s.Pad()
	s.Write([]byte("},\n"))
}

func dumpHttpRequest(s State, v reflect.Value) {
	req := v.Interface().(http.Request)

	s.DumpStructField("URL", reflect.ValueOf(req.URL.String()))
	for _, f := range []string{"Method", "Proto", "ContentLength"} {
		s.DumpStructField(f, v.FieldByName(f))
	}

	dumpHttpHeaders(s, req.Header)

	if req.ContentLength == 0 {
		s.DumpStructField("Body", reflect.ValueOf(""))
	} else if ct := req.Header.Get("Content-Type"); isContentTypeTextSafe(ct) {
		Body, body, err := copyBody(req.Body)
		if err != nil {
			return
		}

		v.FieldByName("Body").Set(reflect.ValueOf(Body))
		s.DumpStructField("Body", reflect.ValueOf(body))
	} else {
		s.DumpStructField("Body", reflect.ValueOf("<BINARY>"))
	}

}

func dumpHttpResponse(s State, v reflect.Value) {
	for _, f := range []string{"Status", "StatusCode", "Proto", "TransferEncoding", "ContentLength"} {
		s.DumpStructField(f, v.FieldByName(f))
	}

	resp := v.Interface().(http.Response)

	dumpHttpHeaders(s, resp.Header)

	if chunked(resp.TransferEncoding) && resp.ContentLength == -1 {
		s.AddComment("streamed")
		s.DumpStructField("Body", reflect.ValueOf(nil))
		return
	}

	if resp.ContentLength == 0 {
		s.DumpStructField("Body", reflect.ValueOf(""))
	} else if ct := resp.Header.Get("Content-Type"); isContentTypeTextSafe(ct) {
		Body, body, err := copyBody(resp.Body)
		if err != nil {
			s.DumpStructField("Body", reflect.ValueOf(nil))
			return
		}

		v.FieldByName("Body").Set(reflect.ValueOf(Body))
		s.DumpStructField("Body", reflect.ValueOf(body))
	} else {
		s.DumpStructField("Body", reflect.ValueOf("<BINARY>"))
	}
}

func isContentTypeTextSafe(ct string) bool {
	if strings.HasPrefix(ct, "text/") {
		return true
	}

	if strings.HasSuffix(ct, "/json") {
		return true
	}

	if strings.HasSuffix(ct, "/xml") {
		return true
	}

	if ct == "application/x-www-form-urlencoded" {
		return true
	}

	if ct == "application/ld+json" || ct == "application/xhtml+xml" {
		return true
	}

	return false
}

// Checks whether chunked is part of the encodings stack
func chunked(te []string) bool { return len(te) > 0 && te[0] == "chunked" }

// copyBody reads all of b to memory and then returns two equivalent
// ReadClosers yielding the same bytes.
//
// It returns an error if the initial slurp of all bytes fails. It does not attempt
// to make the returned ReadClosers have identical error-matching behavior.
func copyBody(b io.ReadCloser) (io.ReadCloser, interface{}, error) {
	if b == http.NoBody || b == nil {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(b); err != nil {
		return b, "<invalid>", err
	}
	if err := b.Close(); err != nil {
		return b, "<invalid>", err
	}

	return ioutil.NopCloser(&buf), buf.String(), nil
}
