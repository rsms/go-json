# json

[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/rsms/go-json.svg)][godoc]
[![PkgGoDev](https://pkg.go.dev/badge/github.com/rsms/go-json)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/rsms/go-json)](https://goreportcard.com/report/github.com/rsms/go-json)

[godoc]: https://pkg.go.dev/github.com/rsms/go-json

Low level, caller-driven JSON builder and parser

Example:

```go
import (
  "fmt"
  "github.com/rsms/go-json"
)

func ExampleBuilder() {
  var b json.Builder
  b.Indent = "  " // enable pretty-printing

  b.StartObject()
    b.Key("subject")
    b.Str("Fun")

    b.Key("labels")
    b.StartArray()
      b.Str("casual")
      b.Str("message")
    b.EndArray()

    // Add base-64 encoded data
    data := []byte("hello world")
    b.StartObjectProp("attachment")
      b.IntProp("size", int64(len(data)), 64)
      b.StrProp("type", "text/plain")
      b.BlobProp("data", data)
    b.EndObject()

    b.BoolProp("isUnread", true)
    b.FloatProp("score", 0.41, 64)
  b.EndObject()

  fmt.Println(string(b.Bytes()))
  // Output:
  // {
  //   "subject": "Fun",
  //   "labels": [
  //     "casual",
  //     "message"
  //   ],
  //   "attachment": {
  //     "size": "11",
  //     "type": "text/plain",
  //     "data": "aGVsbG8gd29ybGQ"
  //   },
  //   "isUnread": true,
  //   "score": 0.41
  // }
}
```
