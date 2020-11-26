package json

// The WriteJsonString function, the jsonSafeSet data set, and
// the WriteJsonFloat function comes from the Go source code
// (encoding/json) and is licensed as follows:
//
// Copyright (c) 2009 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.

// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"unicode/utf8"
)

var hexdigits = "0123456789abcdef"

const intSize int = 32 << (^uint(0) >> 63) // bits of int on target platform

type builderState int

const (
	builderInit  = builderState(iota)
	builderObj   // just wrote `{` or `[`
	builderKey   // just wrote `"key":`
	builderValue // just wrote some value, e.g. `3`, `[]`, `{"key":1}` etc.
)

// func ExampleBuilder() {
// 	var b Builder
// 	b.StartObject()
// 	b.EndObject()
// }

// Builder is a low-level JSON builder with a caller-driven API.
// It can generatet both compact JSON as well as pretty-printed output with almost zero overhead.
type Builder struct {
	bytes.Buffer // output JSON

	// Err holds the first error encountered, if any
	Err error

	// pretty-printing
	Indent  string
	KeyTerm []byte // key terminator. Defaults to ":"

	// w         bytes.Buffer     // output JSON
	state     builderState // most recently built thing
	scratch   []byte       // temporary storage for intermediate encoding
	nestdepth int
}

func (e *Builder) setError(err error) {
	if e.Err == nil {
		e.Err = err
	}
}

func (e *Builder) startChunk(nextstate builderState) {
	switch e.state {

	case builderValue:
		e.WriteByte(',')
		e.newLine()

	case builderObj:
		if len(e.Indent) > 0 {
			e.writeNewLine()
		}

	case builderInit:
		if e.KeyTerm == nil {
			if len(e.Indent) > 0 {
				e.KeyTerm = []byte{':', ' '}
			} else {
				e.KeyTerm = []byte{':'}
			}
		}
	}
	e.state = nextstate
}

func (e *Builder) newLine() {
	if e.nestdepth > 0 && len(e.Indent) > 0 {
		e.writeNewLine()
	}
}

func (e *Builder) writeNewLine() {
	e.WriteByte('\n')
	for i := 0; i < e.nestdepth; i++ {
		e.Write([]byte(e.Indent))
	}
}

// Reset resets the Builder so it can be reused. Does not reset Indent.
// If the ByteWriter has a Reset() method, that method is called as well, which is the case
// when the default bytes.Buffer is being used.
func (e *Builder) Reset() {
	e.Buffer.Reset()
	e.Err = nil
	e.state = builderInit
	e.nestdepth = 0
}

// Key writes `"k":`
func (e *Builder) Key(k string) {
	e.KeyBytes([]byte(k))
}

// KeyBytes writes `"k":`
func (e *Builder) KeyBytes(k []byte) {
	e.startChunk(builderKey)
	e.WriteJsonString(k)
	e.Write(e.KeyTerm)
}

// RawKey writes k verbatim without quotes and without escaping.
// Thus, k is expected to be a valid JSON key already.
func (e *Builder) RawKey(k []byte) {
	e.startChunk(builderKey)
	e.Write(k)
	e.Write(e.KeyTerm)
}

// StartObject starts a dictionary. Equivalent to Start('{')
func (e *Builder) StartObject() { e.Start('{') }

// EndObject ends a dictionary. Equivalent to End('}')
func (e *Builder) EndObject() { e.End('}') }

// StartArray starts a dictionary. Equivalent to Start('[')
func (e *Builder) StartArray() { e.Start('[') }

// EndArray ends a dictionary. Equivalent to End(']')
func (e *Builder) EndArray() { e.End(']') }

// Start a dictionary (kind='{') or list (kind='[')
func (e *Builder) Start(kind byte) {
	e.startChunk(builderObj)
	e.WriteByte(kind)
	e.nestdepth++
}

// End a dictionary (kind='}') or list (kind=']')
// If the builder is not inside an object, this method panics.
func (e *Builder) End(kind byte) {
	e.nestdepth--
	if e.state == builderKey {
		// ending object after writing key but no value
		// e.g. StartObject('{'); Key("foo"); EndObject('}') => `{"foo":}`
		e.Err = fmt.Errorf("key without value")
	} else if e.state != builderObj && len(e.Indent) > 0 {
		e.writeNewLine()
	}
	e.WriteByte(kind)
	e.state = builderValue
}

// InObject returns true if EndObject can be safely called
func (e *Builder) InObject() bool {
	return e.nestdepth > 0 && e.state != builderKey
}

var (
	jsonTrue  = []byte("true")
	jsonFalse = []byte("false")
	jsonNull  = []byte("null")
)

func (e *Builder) Raw(b []byte) {
	e.startChunk(builderValue)
	e.Write(b)
}

func (e *Builder) Null() { e.Raw(jsonNull) }

func (e *Builder) Bool(v bool) {
	if v {
		e.Raw(jsonTrue)
	} else {
		e.Raw(jsonFalse)
	}
}

func (e *Builder) Blob(data []byte) {
	e.startChunk(builderValue)
	b64enc := base64.RawStdEncoding
	b64len := b64enc.EncodedLen(len(data))
	var buf []byte
	if b64len < 512 {
		if cap(e.scratch) < b64len {
			e.scratch = make([]byte, b64len)
		}
		buf = e.scratch[:b64len]
	} else {
		buf = make([]byte, b64len)
	}
	b64enc.Encode(buf, data)
	e.WriteJsonString(buf)
}

func (e *Builder) Str(s string) {
	e.startChunk(builderValue)
	e.WriteJsonString([]byte(s))
}

func (e *Builder) StrBytes(s []byte) {
	e.startChunk(builderValue)
	e.WriteJsonString(s)
}

func (e *Builder) Int(v int64, bitsize int) {
	e.startChunk(builderValue)
	if bitsize > 32 {
		e.WriteJsonString([]byte(strconv.FormatInt(v, 10)))
	} else {
		fmt.Fprintf(e, "%d", v)
	}
}

func (e *Builder) Uint(v uint64, bitsize int) {
	e.startChunk(builderValue)
	if bitsize > 32 {
		e.WriteJsonString([]byte(strconv.FormatUint(v, 10)))
	} else {
		fmt.Fprintf(e, "%d", v)
	}
}

// Float writes a float64 number of bits size
func (e *Builder) Float(f float64, bits int) {
	e.startChunk(builderValue)

	if math.IsInf(f, 0) || math.IsNaN(f) {
		e.setError(fmt.Errorf(
			"unsupported float64 value %s",
			strconv.FormatFloat(f, 'g', -1, int(bits))))
	}

	// Convert as if by ES6 number to string conversion.
	// This matches most other JSON generators.
	// See golang.org/issue/6384 and golang.org/issue/14135.
	// Like fmt %g, but the exponent cutoffs are different
	// and exponents themselves are not padded to two digits.
	b := e.scratch[:0]
	abs := math.Abs(f)
	fmt := byte('f')
	// Note: Must use float32 comparisons for underlying float32 value to get precise cutoffs right.
	if abs != 0 {
		if bits == 64 && (abs < 1e-6 || abs >= 1e21) ||
			bits == 32 && (float32(abs) < 1e-6 ||
				float32(abs) >= 1e21) {
			fmt = 'e'
		}
	}
	b = strconv.AppendFloat(b, f, fmt, -1, int(bits))
	if fmt == 'e' {
		// clean up e-09 to e-9
		n := len(b)
		if n >= 4 && b[n-4] == 'e' && b[n-3] == '-' && b[n-2] == '0' {
			b[n-2] = b[n-1]
			b = b[:n-1]
		}
	}

	e.scratch = b
	e.Write(b)
}

func (e *Builder) Any(v interface{}) {
	switch v := v.(type) {
	case bool:
		e.Bool(v)
	case int:
		e.Int(int64(v), intSize)
	case int8:
		e.Int(int64(v), 8)
	case int16:
		e.Int(int64(v), 16)
	case int32:
		e.Int(int64(v), 32)
	case int64:
		e.Int(v, 64)
	case uint:
		e.Uint(uint64(v), intSize)
	case uint8:
		e.Uint(uint64(v), 8)
	case uint16:
		e.Uint(uint64(v), 16)
	case uint32:
		e.Uint(uint64(v), 32)
	case uint64:
		e.Uint(v, 64)
	case float32:
		e.Float(float64(v), 32)
	case float64:
		e.Float(v, 64)
	case string:
		e.Str(v)
	case []byte:
		e.Blob(v)
	default:
		if v == nil {
			e.Null()
		} else if st, ok := v.(interface{ BuildJSON(*Builder) }); ok {
			st.BuildJSON(e)
		} else if st, ok := v.(interface{ MarshalJSON() ([]byte, error) }); ok {
			json, err := st.MarshalJSON()
			if err != nil {
				e.setError(err)
			} else {
				e.Raw(json)
			}
		} else {
			e.startChunk(builderValue)
			enc := json.NewEncoder(&e.Buffer)
			enc.SetIndent("", e.Indent)
			e.setError(enc.Encode(v))
			// Note: I can't figure out how to make json.Encoder not to write a trailing linebreak.
		}
	}
}

// convenience methods for writing key-value properties while building objects

func (e *Builder) NullProp(k string)                        { e.Key(k); e.Null() }
func (e *Builder) BoolProp(k string, v bool)                { e.Key(k); e.Bool(v) }
func (e *Builder) IntProp(k string, v int64, bitsize int)   { e.Key(k); e.Int(v, bitsize) }
func (e *Builder) UintProp(k string, v uint64, bitsize int) { e.Key(k); e.Uint(v, bitsize) }
func (e *Builder) FloatProp(k string, f float64, bits int)  { e.Key(k); e.Float(f, bits) }
func (e *Builder) StrProp(k, v string)                      { e.Key(k); e.Str(v) }
func (e *Builder) BlobProp(k string, v []byte)              { e.Key(k); e.Blob(v) }
func (e *Builder) AnyProp(k string, v interface{})          { e.Key(k); e.Any(v) }
func (e *Builder) StartObjectProp(k string)                 { e.Key(k); e.StartObject() }
func (e *Builder) StartArrayProp(k string)                  { e.Key(k); e.StartArray() }

// WriteJsonString writes a string, quoting and escaping it as needed.
// This is a lower-level primitive; it's not aware of indentation, commas, etc.
func (e *Builder) WriteJsonString(s []byte) {
	e.WriteByte('"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if jsonSafeSet[b] {
				i++
				continue
			}
			if start < i {
				e.Write(s[start:i])
			}
			e.WriteByte('\\')
			switch b {
			case '\\', '"':
				e.WriteByte(b)
			case '\n':
				e.WriteByte('n')
			case '\r':
				e.WriteByte('r')
			case '\t':
				e.WriteByte('t')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				e.Write([]byte(`u00`))
				e.WriteByte(hexdigits[b>>4])
				e.WriteByte(hexdigits[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRune(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				e.Write(s[start:i])
			}
			e.Write([]byte(`\ufffd`))
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				e.Write(s[start:i])
			}
			e.Write([]byte(`\u202`))
			e.WriteByte(hexdigits[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		e.Write(s[start:])
	}
	e.WriteByte('"')
}

// safeSet holds the value true if the ASCII character with the given array
// position can be represented inside a JSON string without any further
// escaping.
//
// All values are true except for the ASCII control characters (0-31), the
// double quote ("), and the backslash character ("\").
//
// See note about source license in documentation for WriteJsonString
var jsonSafeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      true,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      true,
	'=':      true,
	'>':      true,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}
