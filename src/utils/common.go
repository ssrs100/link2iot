package utils

import (
	"os"
)

const (
	APP_BASE_KEY string = "APP_BASE_DIR"
)

var (
	appBaseDir string
)

func GetAppBaseDir() string {
	if len(appBaseDir) > 0 {
		return appBaseDir
	}
	appBaseDir = os.Getenv(APP_BASE_KEY)
	return appBaseDir
}
