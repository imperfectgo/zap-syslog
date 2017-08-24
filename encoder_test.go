// Modifications Copyright (c) 2017 Timon Wong
// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package zapsyslog

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	syslog "github.com/timonwong/go-syslog"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	testEntry = zapcore.Entry{
		Time:    time.Date(2017, 1, 2, 3, 4, 5, 123456789, time.UTC),
		Message: "fake",
		Level:   zap.DebugLevel,
	}
)

func testEncoderConfig() SyslogEncoderConfig {
	return SyslogEncoderConfig{
		MessageKey:     "msg",
		NameKey:        "name",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,

		Hostname: "localhost",
		App:      "encoder_test",
		PID:      9876,
		Facility: syslog.LOG_LOCAL0,
	}
}

func TestJSONEncoderObjectFields(t *testing.T) {
	tests := []struct {
		desc     string
		expected string
		f        func(zapcore.Encoder)
	}{
		{"binary", `"k":"YWIxMg=="`, func(e zapcore.Encoder) { e.AddBinary("k", []byte("ab12")) }},
		{"bool", `"k\\":true`, func(e zapcore.Encoder) { e.AddBool(`k\`, true) }}, // test key escaping once
		{"bool", `"k":true`, func(e zapcore.Encoder) { e.AddBool("k", true) }},
		{"bool", `"k":false`, func(e zapcore.Encoder) { e.AddBool("k", false) }},
		{"byteString", `"k":"v\\"`, func(e zapcore.Encoder) { e.AddByteString(`k`, []byte(`v\`)) }},
		{"byteString", `"k":"v"`, func(e zapcore.Encoder) { e.AddByteString("k", []byte("v")) }},
		{"byteString", `"k":""`, func(e zapcore.Encoder) { e.AddByteString("k", []byte{}) }},
		{"byteString", `"k":""`, func(e zapcore.Encoder) { e.AddByteString("k", nil) }},
		{"complex128", `"k":"1+2i"`, func(e zapcore.Encoder) { e.AddComplex128("k", 1+2i) }},
		{"complex64", `"k":"1+2i"`, func(e zapcore.Encoder) { e.AddComplex64("k", 1+2i) }},
		{"duration", `"k":0.000000001`, func(e zapcore.Encoder) { e.AddDuration("k", 1) }},
		{"float64", `"k":1`, func(e zapcore.Encoder) { e.AddFloat64("k", 1.0) }},
		{"float64", `"k":10000000000`, func(e zapcore.Encoder) { e.AddFloat64("k", 1e10) }},
		{"float64", `"k":"NaN"`, func(e zapcore.Encoder) { e.AddFloat64("k", math.NaN()) }},
		{"float64", `"k":"+Inf"`, func(e zapcore.Encoder) { e.AddFloat64("k", math.Inf(1)) }},
		{"float64", `"k":"-Inf"`, func(e zapcore.Encoder) { e.AddFloat64("k", math.Inf(-1)) }},
		{"float32", `"k":1`, func(e zapcore.Encoder) { e.AddFloat32("k", 1.0) }},
		{"float32", `"k":10000000000`, func(e zapcore.Encoder) { e.AddFloat32("k", 1e10) }},
		{"float32", `"k":"NaN"`, func(e zapcore.Encoder) { e.AddFloat32("k", float32(math.NaN())) }},
		{"float32", `"k":"+Inf"`, func(e zapcore.Encoder) { e.AddFloat32("k", float32(math.Inf(1))) }},
		{"float32", `"k":"-Inf"`, func(e zapcore.Encoder) { e.AddFloat32("k", float32(math.Inf(-1))) }},
		{"int", `"k":42`, func(e zapcore.Encoder) { e.AddInt("k", 42) }},
		{"int64", `"k":42`, func(e zapcore.Encoder) { e.AddInt64("k", 42) }},
		{"int32", `"k":42`, func(e zapcore.Encoder) { e.AddInt32("k", 42) }},
		{"int16", `"k":42`, func(e zapcore.Encoder) { e.AddInt16("k", 42) }},
		{"int8", `"k":42`, func(e zapcore.Encoder) { e.AddInt8("k", 42) }},
		{"string", `"k":"v\\"`, func(e zapcore.Encoder) { e.AddString(`k`, `v\`) }},
		{"string", `"k":"v"`, func(e zapcore.Encoder) { e.AddString("k", "v") }},
		{"string", `"k":""`, func(e zapcore.Encoder) { e.AddString("k", "") }},
		{"time", `"k":1`, func(e zapcore.Encoder) { e.AddTime("k", time.Unix(1, 0)) }},
		{"uint", `"k":42`, func(e zapcore.Encoder) { e.AddUint("k", 42) }},
		{"uint64", `"k":42`, func(e zapcore.Encoder) { e.AddUint64("k", 42) }},
		{"uint32", `"k":42`, func(e zapcore.Encoder) { e.AddUint32("k", 42) }},
		{"uint16", `"k":42`, func(e zapcore.Encoder) { e.AddUint16("k", 42) }},
		{"uint8", `"k":42`, func(e zapcore.Encoder) { e.AddUint8("k", 42) }},
		{"uintptr", `"k":42`, func(e zapcore.Encoder) { e.AddUintptr("k", 42) }},
		{
			desc:     "object (success)",
			expected: `"k":{"loggable":"yes"}`,
			f: func(e zapcore.Encoder) {
				assert.NoError(t, e.AddObject("k", loggable{true}), "Unexpected error calling MarshalLogObject.")
			},
		},
		{
			desc:     "object (error)",
			expected: `"k":{}`,
			f: func(e zapcore.Encoder) {
				assert.Error(t, e.AddObject("k", loggable{false}), "Expected an error calling MarshalLogObject.")
			},
		},
		{
			desc:     "object (with nested array)",
			expected: `"turducken":{"ducks":[{"in":"chicken"},{"in":"chicken"}]}`,
			f: func(e zapcore.Encoder) {
				assert.NoError(
					t,
					e.AddObject("turducken", turducken{}),
					"Unexpected error calling MarshalLogObject with nested ObjectMarshalers and ArrayMarshalers.",
				)
			},
		},
		{
			desc:     "array (with nested object)",
			expected: `"turduckens":[{"ducks":[{"in":"chicken"},{"in":"chicken"}]},{"ducks":[{"in":"chicken"},{"in":"chicken"}]}]`,
			f: func(e zapcore.Encoder) {
				assert.NoError(
					t,
					e.AddArray("turduckens", turduckens(2)),
					"Unexpected error calling MarshalLogObject with nested ObjectMarshalers and ArrayMarshalers.",
				)
			},
		},
		{
			desc:     "array (success)",
			expected: `"k":[true]`,
			f: func(e zapcore.Encoder) {
				assert.NoError(t, e.AddArray(`k`, loggable{true}), "Unexpected error calling MarshalLogArray.")
			},
		},
		{
			desc:     "array (error)",
			expected: `"k":[]`,
			f: func(e zapcore.Encoder) {
				assert.Error(t, e.AddArray("k", loggable{false}), "Expected an error calling MarshalLogArray.")
			},
		},
		{
			desc:     "reflect (success)",
			expected: `"k":{"loggable":"yes"}`,
			f: func(e zapcore.Encoder) {
				assert.NoError(t, e.AddReflected("k", map[string]string{"loggable": "yes"}), "Unexpected error JSON-serializing a map.")
			},
		},
		{
			desc:     "reflect (failure)",
			expected: "",
			f: func(e zapcore.Encoder) {
				assert.Error(t, e.AddReflected("k", noJSON{}), "Unexpected success JSON-serializing a noJSON.")
			},
		},
		{
			desc: "namespace",
			// EncodeEntry is responsible for closing all open namespaces.
			expected: `"outermost":{"outer":{"foo":1,"inner":{"foo":2,"innermost":{`,
			f: func(e zapcore.Encoder) {
				e.OpenNamespace("outermost")
				e.OpenNamespace("outer")
				e.AddInt("foo", 1)
				e.OpenNamespace("inner")
				e.AddInt("foo", 2)
				e.OpenNamespace("innermost")
			},
		},
	}

	for _, tt := range tests {
		assertOutput(t, tt.desc, tt.expected, tt.f)
	}
}

func TestJSONEncoderArrays(t *testing.T) {
	tests := []struct {
		desc     string
		expected string // expect f to be called twice
		f        func(zapcore.ArrayEncoder)
	}{
		{"bool", `[true,true]`, func(e zapcore.ArrayEncoder) { e.AppendBool(true) }},
		{"byteString", `["k","k"]`, func(e zapcore.ArrayEncoder) { e.AppendByteString([]byte("k")) }},
		{"byteString", `["k\\","k\\"]`, func(e zapcore.ArrayEncoder) { e.AppendByteString([]byte(`k\`)) }},
		{"complex128", `["1+2i","1+2i"]`, func(e zapcore.ArrayEncoder) { e.AppendComplex128(1 + 2i) }},
		{"complex64", `["1+2i","1+2i"]`, func(e zapcore.ArrayEncoder) { e.AppendComplex64(1 + 2i) }},
		{"durations", `[0.000000002,0.000000002]`, func(e zapcore.ArrayEncoder) { e.AppendDuration(2) }},
		{"float64", `[3.14,3.14]`, func(e zapcore.ArrayEncoder) { e.AppendFloat64(3.14) }},
		{"float32", `[3.14,3.14]`, func(e zapcore.ArrayEncoder) { e.AppendFloat32(3.14) }},
		{"int", `[42,42]`, func(e zapcore.ArrayEncoder) { e.AppendInt(42) }},
		{"int64", `[42,42]`, func(e zapcore.ArrayEncoder) { e.AppendInt64(42) }},
		{"int32", `[42,42]`, func(e zapcore.ArrayEncoder) { e.AppendInt32(42) }},
		{"int16", `[42,42]`, func(e zapcore.ArrayEncoder) { e.AppendInt16(42) }},
		{"int8", `[42,42]`, func(e zapcore.ArrayEncoder) { e.AppendInt8(42) }},
		{"string", `["k","k"]`, func(e zapcore.ArrayEncoder) { e.AppendString("k") }},
		{"string", `["k\\","k\\"]`, func(e zapcore.ArrayEncoder) { e.AppendString(`k\`) }},
		{"times", `[1,1]`, func(e zapcore.ArrayEncoder) { e.AppendTime(time.Unix(1, 0)) }},
		{"uint", `[42,42]`, func(e zapcore.ArrayEncoder) { e.AppendUint(42) }},
		{"uint64", `[42,42]`, func(e zapcore.ArrayEncoder) { e.AppendUint64(42) }},
		{"uint32", `[42,42]`, func(e zapcore.ArrayEncoder) { e.AppendUint32(42) }},
		{"uint16", `[42,42]`, func(e zapcore.ArrayEncoder) { e.AppendUint16(42) }},
		{"uint8", `[42,42]`, func(e zapcore.ArrayEncoder) { e.AppendUint8(42) }},
		{"uintptr", `[42,42]`, func(e zapcore.ArrayEncoder) { e.AppendUintptr(42) }},
		{
			desc:     "arrays (success)",
			expected: `[[true],[true]]`,
			f: func(arr zapcore.ArrayEncoder) {
				assert.NoError(t, arr.AppendArray(zapcore.ArrayMarshalerFunc(func(inner zapcore.ArrayEncoder) error {
					inner.AppendBool(true)
					return nil
				})), "Unexpected error appending an array.")
			},
		},
		{
			desc:     "arrays (error)",
			expected: `[[true],[true]]`,
			f: func(arr zapcore.ArrayEncoder) {
				assert.Error(t, arr.AppendArray(zapcore.ArrayMarshalerFunc(func(inner zapcore.ArrayEncoder) error {
					inner.AppendBool(true)
					return errors.New("fail")
				})), "Expected an error appending an array.")
			},
		},
		{
			desc:     "objects (success)",
			expected: `[{"loggable":"yes"},{"loggable":"yes"}]`,
			f: func(arr zapcore.ArrayEncoder) {
				assert.NoError(t, arr.AppendObject(loggable{true}), "Unexpected error appending an object.")
			},
		},
		{
			desc:     "objects (error)",
			expected: `[{},{}]`,
			f: func(arr zapcore.ArrayEncoder) {
				assert.Error(t, arr.AppendObject(loggable{false}), "Expected an error appending an object.")
			},
		},
		{
			desc:     "reflect (success)",
			expected: `[{"foo":5},{"foo":5}]`,
			f: func(arr zapcore.ArrayEncoder) {
				assert.NoError(
					t,
					arr.AppendReflected(map[string]int{"foo": 5}),
					"Unexpected an error appending an object with reflection.",
				)
			},
		},
		{
			desc:     "reflect (error)",
			expected: `[]`,
			f: func(arr zapcore.ArrayEncoder) {
				assert.Error(
					t,
					arr.AppendReflected(noJSON{}),
					"Unexpected an error appending an object with reflection.",
				)
			},
		},
	}

	for _, tt := range tests {
		f := func(enc zapcore.Encoder) error {
			return enc.AddArray("array", zapcore.ArrayMarshalerFunc(func(arr zapcore.ArrayEncoder) error {
				tt.f(arr)
				tt.f(arr)
				return nil
			}))
		}
		assertOutput(t, tt.desc, `"array":`+tt.expected, func(enc zapcore.Encoder) {
			err := f(enc)
			assert.NoError(t, err, "Unexpected error adding array to JSON zapcore.Encoder.")
		})
	}
}

func assertOutput(t testing.TB, desc string, expected string, f func(zapcore.Encoder)) {
	enc := NewSyslogEncoder(SyslogEncoderConfig{
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}).(*syslogEncoder)
	f(enc)
	buf, err := enc.EncodeEntry(testEntry, nil)
	if !assert.NoError(t, err) {
		return
	}
	defer buf.Free()

	const bom = "\xef\xbb\xbf"
	output := buf.String()
	i := strings.Index(output, "\xef\xbb\xbf")
	if !assert.Condition(t, func() (success bool) { return i > 0 }, "not a valid syslog output") {
		return
	}

	jsonString := output[i+len(bom):]
	assert.Contains(t, jsonString, expected)
}

// Nested Array- and ObjectMarshalers.
type turducken struct{}

func (t turducken) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	return enc.AddArray("ducks", zapcore.ArrayMarshalerFunc(func(arr zapcore.ArrayEncoder) error {
		for i := 0; i < 2; i++ {
			arr.AppendObject(zapcore.ObjectMarshalerFunc(func(inner zapcore.ObjectEncoder) error {
				inner.AddString("in", "chicken")
				return nil
			}))
		}
		return nil
	}))
}

type turduckens int

func (t turduckens) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	var err error
	tur := turducken{}
	for i := 0; i < int(t); i++ {
		err = multierr.Append(err, enc.AppendObject(tur))
	}
	return err
}

type loggable struct{ bool }

func (l loggable) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if !l.bool {
		return errors.New("can't marshal")
	}
	enc.AddString("loggable", "yes")
	return nil
}

func (l loggable) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	if !l.bool {
		return errors.New("can't marshal")
	}
	enc.AppendBool(true)
	return nil
}

type noJSON struct{}

func (nj noJSON) MarshalJSON() ([]byte, error) {
	return nil, errors.New("no")
}

func TestSyslogEncoder(t *testing.T) {
	enc := NewSyslogEncoder(testEncoderConfig())
	enc.AddString("str", "foo")
	enc.AddInt64("int64-1", 1)
	enc.AddInt64("int64-2", 2)
	enc.AddFloat64("float64", 1.0)
	enc.AddString("string1", "\n")
	enc.AddString("string2", "💩")
	enc.AddString("string3", "🤔")
	enc.AddString("string4", "🙊")
	enc.AddBool("bool", true)
	buf, _ := enc.EncodeEntry(testEntry, nil)
	defer buf.Free()

	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("Wrong syslog output: no line ending")
		return
	}

	expected := "<135>1 2017-01-02T03:04:05.123456Z localhost encoder_test 9876 - - \xef\xbb\xbf"
	if !strings.HasPrefix(output, expected) {
		t.Errorf("Wrong syslog output!")
		t.Logf("output is: %s", output)
		return
	}

	jsonString := output[len(expected):]
	var m map[string]interface{}
	err := json.Unmarshal([]byte(jsonString), &m)
	if err != nil {
		t.Errorf("message part of syslog output is not a valid json string: %s", err)
		t.Logf("json string is: %s", jsonString)
		return
	}
}
