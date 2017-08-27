package main

import (
	"crypto/rand"
	"encoding/hex"
	"os"

	"github.com/timonwong/zap-syslog"
	"github.com/timonwong/zap-syslog/syslog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	enc := zapsyslog.NewSyslogEncoder(zapsyslog.SyslogEncoderConfig{
		EncoderConfig: zapcore.EncoderConfig{
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.EpochTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},

		Facility: syslog.LOG_LOCAL0,
		Hostname: "localhost",
		PID:      os.Getpid(),
		App:      "zapsyslog-test",
	})

	sink, err := zapsyslog.NewConnSyncer("tcp", "localhost:514")
	if err != nil {
		panic(err)
	}

	atom := zap.NewAtomicLevel()
	logger := zap.New(zapcore.NewCore(
		enc,
		zapcore.Lock(sink),
		atom,
	))

	l := logger.With(zap.String("str", "foo"))
	for i := 0; i < 100; i++ {
		l.Info("Hello, world!", zap.Int("int", i), zap.String("hex", generateHexString()))
	}
}

func generateHexString() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
