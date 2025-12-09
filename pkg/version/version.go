package version

import (
	"fmt"
	"runtime"
)

var (
	Version   = "dev"
	Revision  = "unknown"
	Branch    = "unknown"
	BuildUser = "unknown"
	BuildDate = "unknown"
)

// Print returns version information
func Print(name string) string {
	return fmt.Sprintf(`%s version %s
  revision: %s
  branch: %s
  build user: %s
  build date: %s
  go version: %s
  go compiler: %s
  platform: %s/%s`,
		name,
		Version,
		Revision,
		Branch,
		BuildUser,
		BuildDate,
		runtime.Version(),
		runtime.Compiler,
		runtime.GOOS,
		runtime.GOARCH,
	)
}
