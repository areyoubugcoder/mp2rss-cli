// Package cliopts exposes the shared command dependency surface.
//
// Cobra parses persistent flags only after the subcommand binds, so we pass
// getters (not raw strings) through Deps. Subcommands call deps.Output() etc.
// at RunE time to read the resolved value.
package cliopts

// Deps is the dependency surface for subcommands.
type Deps struct {
	// Output returns the resolved --output value (table|json).
	Output func() string
	// APIKey returns the --api-key flag value (may be empty).
	APIKey func() string
	// APIURL returns the --api-url flag value (may be empty).
	APIURL func() string
}
