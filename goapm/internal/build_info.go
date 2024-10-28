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

// BuildInfo is used to get build information about the application.
var BuildInfo = &buildInfo{}

// Hostname returns the hostname of the machine running the application.
func (b *buildInfo) Hostname() string {
	return hostname
}

// AppName returns the name of the application.
func (b *buildInfo) AppName() string {
	return appName
}
