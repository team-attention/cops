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
		newHealthModule(),
		newAuthModule(),
		newDashboardModule(),
		newFeaturedModule(),
		newProjectModule(),
		newRBACModule(),
		newUserModule(),
		newOrganizationModule(),
		newInvitationModule(),
		newAPIKeyModule(),
		newEventModule(),
		newRetryModule(),
		newSessionModule(),

		// Registrations (invoked for side effects)
		fx.Invoke(registerConnectRPCServer),

		// Lifecycle timeouts
		fx.StartTimeout(30*time.Second),
		fx.StopTimeout(30*time.Second),
	).Run()
}
