package betterstack

import "fmt"

// ErrSourceTokenRequired is returned when Config.SourceToken is empty.
type ErrSourceTokenRequired string

func (e ErrSourceTokenRequired) Error() string {
	return fmt.Sprintf("betterstack: source token required (got %q)", string(e))
}
