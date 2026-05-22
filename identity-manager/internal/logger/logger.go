package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

func New(level string) *logrus.Entry {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{})

	parsed, err := logrus.ParseLevel(level)
	if err != nil {
		parsed = logrus.InfoLevel
	}
	log.SetLevel(parsed)

	return log.WithField("app", "identity-manager")
}
