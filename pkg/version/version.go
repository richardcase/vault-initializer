package version

import "github.com/golang/glog"

// Version is the version number of the initializer
var Version string

// BuildDate is the date the application was built
var BuildDate string

// GitHash is the Git commit hash that was used when building the release
var GitHash string

// OutputVersion writes the version information to the log
func OutputVersion() {
	glog.Info("Version: %s\n", Version)
	glog.Info("Buld Date: %s\n", BuildDate)
	glog.Info("Git Hash: %s\n", GitHash)
}
