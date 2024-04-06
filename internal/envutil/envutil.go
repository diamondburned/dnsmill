package envutil

import (
	"fmt"
	"os"
)

// AssertEnv asserts that the given environment variables are set.
func AssertEnv(envs ...string) error {
	for _, env := range envs {
		if v := os.Getenv(env); v == "" {
			return fmt.Errorf("environment variable %s is not set", env)
		}
	}
	return nil
}
