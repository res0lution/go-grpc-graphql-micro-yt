package template

import "fmt"

func errResolverNotConfigured(name string) error {
	return fmt.Errorf("%s resolver is not configured", name)
}
