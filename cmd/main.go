package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/alecthomas/kingpin/v2"
	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"

	sketchCmd "github.com/bluemir/paint/cmd/sketch"
	"github.com/bluemir/paint/internal/buildinfo"
)

const (
	describe        = ``
	defaultLogLevel = logrus.WarnLevel
)

func Run() error {
	conf := struct {
		logLevel  int
		logFormat string
	}{}

	app := kingpin.New(buildinfo.AppName, describe)
	app.Version(buildinfo.Version + "\nbuildtime:" + buildinfo.BuildTime + "\nmode:" + buildinfo.BuildMode)

	app.Flag("verbose", "Log level").
		Short('v').
		CounterVar(&conf.logLevel)
	app.Flag("log-format", "Log format").
		StringVar(&conf.logFormat)
	app.PreAction(func(*kingpin.ParseContext) error {
		level := defaultLogLevel + logrus.Level(conf.logLevel) // #nosec G115 - CLI counter is non-negative
		if level > logrus.TraceLevel {
			level = logrus.TraceLevel
		}
		logrus.SetOutput(os.Stderr)
		logrus.SetLevel(level)
		logrus.SetReportCaller(true)
		logrus.Infof("logrus level: %s", level)

		callerPrettyfier := func(f *runtime.Frame) (string, string) {
			/* https://github.com/sirupsen/logrus/issues/63#issuecomment-476486166 */
			return "", fmt.Sprintf("%s:%d", f.File, f.Line)
		}

		switch conf.logFormat {
		case "text-color":
			logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true, CallerPrettyfier: callerPrettyfier})
		case "json":
			logrus.SetFormatter(&logrus.JSONFormatter{CallerPrettyfier: callerPrettyfier})
		case "", "text":
			logrus.StandardLogger().Formatter = &logrus.TextFormatter{CallerPrettyfier: callerPrettyfier}
		default:
			return errors.Errorf("unknown log format")
		}

		return nil
	})

	sketchCmd.Register(app.Command("sketch", "칸 단위로 글자와 256색을 칠하는 터미널 그림판").Default())

	cmd, err := app.Parse(os.Args[1:])
	if err != nil {
		return err
	}
	logrus.Debug(cmd)
	return nil
}
