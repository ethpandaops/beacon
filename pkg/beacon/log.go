package beacon

import (
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
)

func (n *node) GetZeroLogLevel() zerolog.Level {
	if n.log == nil {
		return zerolog.NoLevel
	}

	var logLevel logrus.Level

	// Handle both Logger and Entry types
	switch v := n.log.(type) {
	case *logrus.Logger:
		logLevel = v.GetLevel()
	case *logrus.Entry:
		logLevel = v.Logger.GetLevel()
	default:
		return zerolog.NoLevel
	}

	zerologLevel := zerolog.NoLevel

	switch logLevel {
	case logrus.DebugLevel:
		zerologLevel = zerolog.DebugLevel
	case logrus.InfoLevel:
		zerologLevel = zerolog.InfoLevel
	case logrus.WarnLevel:
		zerologLevel = zerolog.WarnLevel
	case logrus.ErrorLevel:
		zerologLevel = zerolog.ErrorLevel
	case logrus.FatalLevel:
		zerologLevel = zerolog.FatalLevel
	case logrus.PanicLevel:
		zerologLevel = zerolog.PanicLevel
	}

	return zerologLevel
}
