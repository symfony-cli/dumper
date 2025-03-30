//go:build go1.23

package dumper

const httpRequestExceptedDump = `http.Request{
  Method: "",
  URL: nil, // &url.URL
  Proto: "",
  ProtoMajor: 0,
  ProtoMinor: 0,
  Header: nil, // map[string][]string
  Body: nil,
  GetBody: nil, // func() (io.ReadCloser, error)
  ContentLength: 0, // int64
  TransferEncoding: nil, // []string
  Close: false,
  Host: "",
  Form: nil, // map[string][]string
  PostForm: nil, // map[string][]string
  MultipartForm: nil, // &multipart.Form
  Trailer: nil, // map[string][]string
  RemoteAddr: "",
  RequestURI: "",
  TLS: nil, // &tls.ConnectionState
  Cancel: nil, // <-chan struct {}
  Response: nil, // &http.Response
  Pattern: "",
}`

const httpRequestExceptedDumpWithPrivateFields = `http.Request{
  Method: "",
  URL: nil, // &url.URL
  Proto: "",
  ProtoMajor: 0,
  ProtoMinor: 0,
  Header: nil, // map[string][]string
  Body: nil,
  GetBody: nil, // func() (io.ReadCloser, error)
  ContentLength: 0, // int64
  TransferEncoding: nil, // []string
  Close: false,
  Host: "",
  Form: nil, // map[string][]string
  PostForm: nil, // map[string][]string
  MultipartForm: nil, // &multipart.Form
  Trailer: nil, // map[string][]string
  RemoteAddr: "",
  RequestURI: "",
  TLS: nil, // &tls.ConnectionState
  Cancel: nil, // <-chan struct {}
  Response: nil, // &http.Response
  Pattern: "",
  ctx: nil,
  pat: nil, // &http.pattern
  matches: nil, // []string
  otherValues: nil, // map[string]string
}`
