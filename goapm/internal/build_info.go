package internal

import (
	"os"
	"path/filepath"
)

var (
	hostname string
	appName  string
)

func init() {
	hostname, _ = os.Hostname()
	appName = os.Getenv("APP_NAME")
	if appName == "" {
		appName = filepath.Base(os.Args[0])
	}
}

type buildInfo struct{}

var BuildInfo = &buildInfo{}

func (b *buildInfo) Hostname() string {
	return hostname
}

func (b *buildInfo) AppName() string {
	return appName
}
