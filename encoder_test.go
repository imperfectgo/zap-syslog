// Copyright (c) 2017 Timon Wong
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
	"log/syslog"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	buf, _ := enc.EncodeEntry(zapcore.Entry{
		Time:    time.Date(2017, 1, 2, 3, 4, 5, 123456789, time.UTC),
		Message: "fake",
		Level:   zap.DebugLevel,
	}, nil)
	defer buf.Free()

	output := buf.String()
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
