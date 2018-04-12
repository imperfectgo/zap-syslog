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
	"os"
	"path"
	"strings"
	"time"

	"github.com/imperfectgo/zap-syslog/internal"
	"github.com/imperfectgo/zap-syslog/internal/bufferpool"
	"github.com/imperfectgo/zap-syslog/syslog"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const (
	version         = 1
	severityMask    = 0x07
	facilityMask    = 0xf8
	nilValue        = "-"
	timestampFormat = "2006-01-02T15:04:05.000000Z07:00" // RFC3339 with micro fraction seconds
	maxHostnameLen  = 255
	maxAppNameLen   = 48
)

var (
	_ zapcore.Encoder = &syslogEncoder{}
	_                 = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()).(jsonEncoder)
)

// Framing.
const (
	NonTransparentFraming Framing = iota
	OctetCountingFraming
	DefaultFraming = NonTransparentFraming
)

// Framing configures RFC6587 TCP transport framing.
type Framing int

type jsonEncoder interface {
	zapcore.Encoder
	zapcore.ArrayEncoder
}

// SyslogEncoderConfig allows users to configure the concrete encoders for zap syslog.
type SyslogEncoderConfig struct {
	zapcore.EncoderConfig

	Framing  Framing         `json:"framing" yaml:"framing"`
	Facility syslog.Priority `json:"facility" yaml:"facility"`
	Hostname string          `json:"hostname" yaml:"hostname"`
	PID      int             `json:"pid" yaml:"pid"`
	App      string          `json:"app" yaml:"app"`
}

type syslogEncoder struct {
	*SyslogEncoderConfig
	je jsonEncoder
}

func rfc5424CompliantASCIIMapper(r rune) rune {
	// PRINTUSASCII    = %d33-126
	if r < 33 || r > 126 {
		return '_'
	}
	return r
}

func toRFC5424CompliantASCIIString(s string) string {
	return strings.Map(rfc5424CompliantASCIIMapper, s)
}

// NewSyslogEncoder creates a syslogEncoder.
func NewSyslogEncoder(cfg SyslogEncoderConfig) zapcore.Encoder {
	if cfg.Hostname == "" {
		hostname, _ := os.Hostname()
		cfg.Hostname = hostname
	}
	if cfg.Hostname == "" {
		cfg.Hostname = nilValue
	} else {
		hostname := toRFC5424CompliantASCIIString(cfg.Hostname)
		if len(hostname) > maxHostnameLen {
			hostname = hostname[:maxHostnameLen]
		}
		cfg.Hostname = hostname
	}

	if cfg.PID == 0 {
		cfg.PID = os.Getpid()
	}
	if cfg.App == "" {
		cfg.App = nilValue
	} else {
		app := cfg.App
		if len(app) > maxAppNameLen {
			app = path.Base(app)
		}
		if len(app) > maxAppNameLen {
			app = app[:maxAppNameLen]
		}
		app = toRFC5424CompliantASCIIString(app)
	}

	cfg.EncoderConfig.LineEnding = "\n"
	je := zapcore.NewJSONEncoder(cfg.EncoderConfig).(jsonEncoder)
	return &syslogEncoder{
		SyslogEncoderConfig: &cfg,
		je:                  je,
	}
}

func (enc *syslogEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	return enc.je.AddArray(key, arr)
}

func (enc *syslogEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	return enc.je.AddObject(key, obj)
}

func (enc *syslogEncoder) AddBinary(key string, val []byte)          { enc.je.AddBinary(key, val) }
func (enc *syslogEncoder) AddByteString(key string, val []byte)      { enc.je.AddByteString(key, val) }
func (enc *syslogEncoder) AddBool(key string, val bool)              { enc.je.AddBool(key, val) }
func (enc *syslogEncoder) AddComplex128(key string, val complex128)  { enc.je.AddComplex128(key, val) }
func (enc *syslogEncoder) AddDuration(key string, val time.Duration) { enc.je.AddDuration(key, val) }
func (enc *syslogEncoder) AddFloat64(key string, val float64)        { enc.je.AddFloat64(key, val) }
func (enc *syslogEncoder) AddInt64(key string, val int64)            { enc.je.AddInt64(key, val) }

func (enc *syslogEncoder) AddReflected(key string, obj interface{}) error {
	return enc.je.AddReflected(key, obj)
}

func (enc *syslogEncoder) OpenNamespace(key string)          { enc.je.OpenNamespace(key) }
func (enc *syslogEncoder) AddString(key, val string)         { enc.je.AddString(key, val) }
func (enc *syslogEncoder) AddTime(key string, val time.Time) { enc.je.AddTime(key, val) }
func (enc *syslogEncoder) AddUint64(key string, val uint64)  { enc.je.AddUint64(key, val) }

func (enc *syslogEncoder) AppendArray(arr zapcore.ArrayMarshaler) error {
	return enc.je.AppendArray(arr)
}

func (enc *syslogEncoder) AppendObject(obj zapcore.ObjectMarshaler) error {
	return enc.je.AppendObject(obj)
}

func (enc *syslogEncoder) AppendBool(val bool)              { enc.je.AppendBool(val) }
func (enc *syslogEncoder) AppendByteString(val []byte)      { enc.je.AppendByteString(val) }
func (enc *syslogEncoder) AppendComplex128(val complex128)  { enc.je.AppendComplex128(val) }
func (enc *syslogEncoder) AppendDuration(val time.Duration) { enc.je.AppendDuration(val) }
func (enc *syslogEncoder) AppendInt64(val int64)            { enc.je.AppendInt64(val) }

func (enc *syslogEncoder) AppendReflected(val interface{}) error {
	return enc.je.AppendReflected(val)
}

func (enc *syslogEncoder) AppendString(val string)            { enc.je.AppendString(val) }
func (enc *syslogEncoder) AppendTime(val time.Time)           { enc.je.AppendTime(val) }
func (enc *syslogEncoder) AppendUint64(val uint64)            { enc.je.AppendUint64(val) }
func (enc *syslogEncoder) AddComplex64(k string, v complex64) { enc.je.AddComplex64(k, v) }
func (enc *syslogEncoder) AddFloat32(k string, v float32)     { enc.je.AddFloat32(k, v) }
func (enc *syslogEncoder) AddInt(k string, v int)             { enc.je.AddInt(k, v) }
func (enc *syslogEncoder) AddInt32(k string, v int32)         { enc.je.AddInt32(k, v) }
func (enc *syslogEncoder) AddInt16(k string, v int16)         { enc.je.AddInt16(k, v) }
func (enc *syslogEncoder) AddInt8(k string, v int8)           { enc.je.AddInt8(k, v) }
func (enc *syslogEncoder) AddUint(k string, v uint)           { enc.je.AddUint(k, v) }
func (enc *syslogEncoder) AddUint32(k string, v uint32)       { enc.je.AddUint32(k, v) }
func (enc *syslogEncoder) AddUint16(k string, v uint16)       { enc.je.AddUint16(k, v) }
func (enc *syslogEncoder) AddUint8(k string, v uint8)         { enc.je.AddUint8(k, v) }
func (enc *syslogEncoder) AddUintptr(k string, v uintptr)     { enc.je.AddUintptr(k, v) }
func (enc *syslogEncoder) AppendComplex64(v complex64)        { enc.je.AppendComplex64(v) }
func (enc *syslogEncoder) AppendFloat64(v float64)            { enc.je.AppendFloat64(v) }
func (enc *syslogEncoder) AppendFloat32(v float32)            { enc.je.AppendFloat32(v) }
func (enc *syslogEncoder) AppendInt(v int)                    { enc.je.AppendInt(v) }
func (enc *syslogEncoder) AppendInt32(v int32)                { enc.je.AppendInt32(v) }
func (enc *syslogEncoder) AppendInt16(v int16)                { enc.je.AppendInt16(v) }
func (enc *syslogEncoder) AppendInt8(v int8)                  { enc.je.AppendInt8(v) }
func (enc *syslogEncoder) AppendUint(v uint)                  { enc.je.AppendUint(v) }
func (enc *syslogEncoder) AppendUint32(v uint32)              { enc.je.AppendUint32(v) }
func (enc *syslogEncoder) AppendUint16(v uint16)              { enc.je.AppendUint16(v) }
func (enc *syslogEncoder) AppendUint8(v uint8)                { enc.je.AppendUint8(v) }
func (enc *syslogEncoder) AppendUintptr(v uintptr)            { enc.je.AppendUintptr(v) }

func (enc *syslogEncoder) Clone() zapcore.Encoder {
	return enc.clone()
}

func (enc *syslogEncoder) clone() *syslogEncoder {
	clone := &syslogEncoder{
		SyslogEncoderConfig: enc.SyslogEncoderConfig,
		je:                  enc.je.Clone().(jsonEncoder),
	}
	return clone
}

func (enc *syslogEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	msg := bufferpool.Get()

	var p syslog.Priority
	switch ent.Level {
	case zapcore.FatalLevel:
		p = syslog.LOG_EMERG
	case zapcore.PanicLevel:
		p = syslog.LOG_CRIT
	case zapcore.DPanicLevel:
		p = syslog.LOG_CRIT
	case zapcore.ErrorLevel:
		p = syslog.LOG_ERR
	case zapcore.WarnLevel:
		p = syslog.LOG_WARNING
	case zapcore.InfoLevel:
		p = syslog.LOG_INFO
	case zapcore.DebugLevel:
		p = syslog.LOG_DEBUG
	}
	pr := int64((enc.Facility & facilityMask) | (p & severityMask))

	// <PRI>version
	msg.AppendByte('<')
	msg.AppendInt(pr)
	msg.AppendByte('>')
	msg.AppendInt(version)

	// SP TIMESTAMP
	msg.AppendByte(' ')
	if ent.Time.IsZero() {
		msg.AppendString(nilValue)
	} else {
		msg.AppendString(ent.Time.Format(timestampFormat))
	}

	// SP HOSTNAME
	msg.AppendByte(' ')
	msg.AppendString(enc.Hostname)

	// SP APP-NAME
	msg.AppendByte(' ')
	msg.AppendString(enc.App)

	// SP PROCID
	msg.AppendByte(' ')
	msg.AppendInt(int64(enc.PID))

	// SP MSGID SP STRUCTURED-DATA (just ignore)
	msg.AppendString(" - -")

	// SP UTF8 MSG
	json, err := enc.je.EncodeEntry(ent, fields)
	if json.Len() > 0 {
		msg.AppendString(" \xef\xbb\xbf")
		bs := json.Bytes()
		if enc.Framing == OctetCountingFraming {
			// Strip trailing line feed
			bs = bs[:len(bs)-1]
		}
		msg.AppendString(internal.BytesToString(bs))
	}

	if enc.Framing != OctetCountingFraming {
		return msg, err
	}

	// SYSLOG-FRAME = MSG-LEN SP SYSLOG-MSG
	out := bufferpool.Get()
	out.AppendInt(int64(msg.Len()))
	out.AppendByte(' ')
	out.AppendString(internal.BytesToString(msg.Bytes()))
	msg.Free()
	return out, err
}
