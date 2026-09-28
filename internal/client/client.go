package client

import (
	"github.com/sirupsen/logrus"
)

type Config struct {
	Endpoint string
}

func Run(conf *Config) error {
	logrus.Info("hello world")
	return nil
}
