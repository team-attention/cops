package container

import "go.uber.org/dig"

// Run creates the container, registers all providers, and executes the root command.
func Run() error {
	c := dig.New()

	// Register modules (providers + group registrations)
	// Note: Auth module must be registered before platform module
	// because platform's AuthStatePort depends on auth's AuthAPIPort
	if err := newAuthModule(c); err != nil {
		return err
	}
	if err := newPlatformModule(c); err != nil {
		return err
	}
	if err := newUserModule(c); err != nil {
		return err
	}
	if err := newTrackingModule(c); err != nil {
		return err
	}
	if err := newDaemonModule(c); err != nil {
		return err
	}

	// Register and execute commands (trigger)
	return RegisterCobraCommands(c)
}
