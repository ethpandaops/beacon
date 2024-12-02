package beacon_test

import (
	"testing"

	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGetZeroLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		logLevel logrus.Level
		want     zerolog.Level
	}{
		{
			name:     "debug level",
			logLevel: logrus.DebugLevel,
			want:     zerolog.DebugLevel,
		},
		{
			name:     "info level",
			logLevel: logrus.InfoLevel,
			want:     zerolog.InfoLevel,
		},
		{
			name:     "warn level",
			logLevel: logrus.WarnLevel,
			want:     zerolog.WarnLevel,
		},
		{
			name:     "error level",
			logLevel: logrus.ErrorLevel,
			want:     zerolog.ErrorLevel,
		},
		{
			name:     "fatal level",
			logLevel: logrus.FatalLevel,
			want:     zerolog.FatalLevel,
		},
		{
			name:     "panic level",
			logLevel: logrus.PanicLevel,
			want:     zerolog.PanicLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetLevel(tt.logLevel)

			node := beacon.NewNode(logger, &beacon.Config{}, "", beacon.Options{})
			got := node.GetZeroLogLevel()

			assert.Equal(t, tt.want, got)
		})
	}
}
