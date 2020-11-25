package json

import (
	// "testing"
	"fmt"
	"time"
)

func ExampleBuilder() {
	var b Builder
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
	// data, _ := ioutil.ReadFile("builder_test.go")
	data := []byte("hello world")
	b.StartObjectProp("attachment")
	b.IntProp("size", int64(len(data)), 64)
	b.StrProp("type", "text/plain")
	b.BlobProp("data", data)
	b.EndObject()

	b.BoolProp("isUnread", true)
	b.FloatProp("score", 0.41, 64)

	b.AnyProp("any.bool", true)
	b.AnyProp("any.int8", int8(123))
	b.AnyProp("any.uint8", uint8(123))
	b.AnyProp("any.int16", int16(123))
	b.AnyProp("any.uint16", uint16(123))
	b.AnyProp("any.int32", int32(123))
	b.AnyProp("any.uint32", uint32(123))
	b.AnyProp("any.int64", int64(123))
	b.AnyProp("any.uint64", uint64(123))
	b.AnyProp("any.float32", float32(1.23))
	b.AnyProp("any.float64", float64(1.23))
	b.AnyProp("any.string", "ett två tre")
	b.AnyProp("any.blob", []byte("un dos tres"))
	b.AnyProp("any.custom", time.Unix(123, 0)) // uses MarshalJSON or encoding/json.Encoder

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
	//   "score": 0.41,
	//   "any.bool": true,
	//   "any.int8": 123,
	//   "any.uint8": 123,
	//   "any.int16": 123,
	//   "any.uint16": 123,
	//   "any.int32": 123,
	//   "any.uint32": 123,
	//   "any.int64": "123",
	//   "any.uint64": "123",
	//   "any.float32": 1.23,
	//   "any.float64": 1.23,
	//   "any.string": "ett två tre",
	//   "any.blob": "dW4gZG9zIHRyZXM",
	//   "any.custom": "1969-12-31T16:02:03-08:00"
	// }
}
