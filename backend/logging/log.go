package logging

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"time"
)

var logger = logrus.Logger{
	Out: os.Stderr,
	Formatter: &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "time",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "msg",
			logrus.FieldKeyFunc:  "func",
		},
		TimestampFormat: time.RFC3339Nano,
	},
	Hooks:        make(logrus.LevelHooks),
	Level:        logrus.DebugLevel,
	ReportCaller: false,
}

func init() {
	gin.DefaultWriter = logger.WriterLevel(logrus.DebugLevel)
	gin.DefaultErrorWriter = logger.WriterLevel(logrus.ErrorLevel)
	logger.Level = logrus.InfoLevel
	log.SetOutput(logger.Writer())
}

func Errorf(format string, v ...interface{}) {
	logger.Errorf(format, v...)
}

func Infof(format string, v ...interface{}) {
	logger.Infof(format, v...)
}

func Debugf(format string, v ...interface{}) {
	logger.Debugf(format, v...)
}
