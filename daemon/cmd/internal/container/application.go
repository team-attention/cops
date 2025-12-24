package container

import (
	"time"

	"go.uber.org/fx"
)

// Run creates and runs the fx application.
func Run() {
	fx.New(
		// Modules
		newPlatformModule(),
		newConfigModule(),
		newLogModule(),

		// Handler registration
		fx.Invoke(registerFsnotify),

		// Lifecycle timeouts
		fx.StartTimeout(30*time.Second),
		fx.StopTimeout(30*time.Second),
	).Run()
}
