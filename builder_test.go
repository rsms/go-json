package json

import (
	// "testing"
	"fmt"
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
