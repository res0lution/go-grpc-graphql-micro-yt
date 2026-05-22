package logger

import (
 "os"
 "runtime/debug"

 "github.com/sirupsen/logrus"
)

var log = logrus.New()

func Init(level string) {
 log.SetOutput(os.Stdout)

 log.SetFormatter(&logrus.TextFormatter{
  ForceColors:   true,
  FullTimestamp: true,
 })

 lvl, err := logrus.ParseLevel(level)
 if err != nil {
  lvl = logrus.InfoLevel
 }

 log.SetLevel(lvl)
}

func L() *logrus.Entry {
 return log.WithField("app", "portal-core")
}

func WithComponent(name string) *logrus.Entry {
 return L().WithField("component", name)
}

func GetStack() string {
 return string(debug.Stack())
}