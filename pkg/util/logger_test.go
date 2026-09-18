//go:build !race
// +build !race

package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildLogger(t *testing.T) {
	asserts := assert.New(t)
	asserts.NotPanics(func() {
		BuildLogger("error")
	})
	asserts.NotPanics(func() {
		BuildLogger("warning")
	})
	asserts.NotPanics(func() {
		BuildLogger("info")
	})
	asserts.NotPanics(func() {
		BuildLogger("?")
	})
	asserts.NotPanics(func() {
		BuildLogger("debug")
	})
}

func TestLog(t *testing.T) {
	asserts := assert.New(t)
	asserts.NotNil(Log())
	GloablLogger = nil
	asserts.NotNil(Log())
}

func TestLogger_Debug(t *testing.T) {
	asserts := assert.New(t)
	l := Logger{
		level: LevelDebug,
	}
	asserts.NotPanics(func() {
		l.Debug("123")
	})
	l.level = LevelError
	asserts.NotPanics(func() {
		l.Debug("123")
	})
}

func TestLogger_Info(t *testing.T) {
	asserts := assert.New(t)
	l := Logger{
		level: LevelDebug,
	}
	asserts.NotPanics(func() {
		l.Info("123")
	})
	l.level = LevelError
	asserts.NotPanics(func() {
		l.Info("123")
	})
}
func TestLogger_Warning(t *testing.T) {
	asserts := assert.New(t)
	l := Logger{
		level: LevelDebug,
	}
	asserts.NotPanics(func() {
		l.Warning("123")
	})
	l.level = LevelError
	asserts.NotPanics(func() {
		l.Warning("123")
	})
}

func TestLogger_Error(t *testing.T) {
	asserts := assert.New(t)
	l := Logger{
		level: LevelDebug,
	}
	asserts.NotPanics(func() {
		l.Error("123")
	})
	l.level = -1
	asserts.NotPanics(func() {
		l.Error("123")
	})
}

func TestLogger_Panic(t *testing.T) {
	asserts := assert.New(t)
	l := Logger{
		level: LevelDebug,
	}
	asserts.Panics(func() {
		l.Panic("123")
	})
	l.level = -1
	asserts.NotPanics(func() {
		l.Error("123")
	})
}
