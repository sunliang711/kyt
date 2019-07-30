package utils

import (
	"github.com/sirupsen/logrus"
	"strings"
)

var logLvl = map[string]logrus.Level{
	"debug": logrus.DebugLevel,
	"trace": logrus.TraceLevel,
	"info":  logrus.InfoLevel,
	"warn":  logrus.WarnLevel,
	"error": logrus.ErrorLevel,
	"fatal": logrus.FatalLevel,
	"panic": logrus.PanicLevel,
}

func LogLevel(lvlStr string) logrus.Level {
	lvl := strings.ToLower(lvlStr)
	if l, ok := logLvl[lvl]; ok {
		return l
	}
	return logrus.InfoLevel
}
