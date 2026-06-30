package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds the resolved runtime configuration for the teaming command.
type Config struct {
	ConfigFile string
	Input      string
	Output     string
	Min        int
	Max        int
}

// bindFlags wires each cobra flag to its matching viper key and sets up the
// environment-variable mapping. Call this after all flags are defined on cmd.
//
// Precedence (highest to lowest):
//  1. command-line flags
//  2. environment variables (TEAMING_INPUT, TEAMING_OUTPUT, TEAMING_MIN, TEAMING_MAX)
//  3. config file (teaming.yaml in working directory, or --config path)
//  4. flag defaults
func bindFlags(v *viper.Viper, cmd *cobra.Command) error {
	v.SetEnvPrefix("TEAMING")
	v.AutomaticEnv()

	for _, name := range []string{"input", "output", "min", "max"} {
		if err := v.BindPFlag(name, cmd.Flags().Lookup(name)); err != nil {
			return fmt.Errorf("teaming: binding flag %q: %w", name, err)
		}
	}

	return nil
}

// loadConfigFile adds the config file to v. If configFile is non-empty it is
// used directly; otherwise v searches the working directory for teaming.yaml.
// A missing file is not an error.
func loadConfigFile(v *viper.Viper, configFile string) error {
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName("teaming")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil // missing file is not an error
		}
		// When SetConfigFile is used, viper does not wrap the error in
		// ConfigFileNotFoundError even for a missing path, so also tolerate
		// "no such file" style errors when a file was not explicitly provided.
		if configFile == "" {
			return nil
		}
		return fmt.Errorf("teaming: reading config file: %w", err)
	}

	return nil
}

// Resolve builds a Config by reading values from v (which already has flags
// bound and the config file loaded).
func Resolve(v *viper.Viper) Config {
	return Config{
		Input:  v.GetString("input"),
		Output: v.GetString("output"),
		Min:    v.GetInt("min"),
		Max:    v.GetInt("max"),
	}
}

// Validate checks the config for logical errors and applies defaults. It
// mutates cfg in place (Output defaults to Input for in-place operation) and
// returns the first error encountered.
func (cfg *Config) Validate() error {
	if cfg.Input == "" {
		return fmt.Errorf("teaming: --input is required")
	}

	// Default output to input (in-place regeneration).
	if cfg.Output == "" {
		cfg.Output = cfg.Input
	}

	if cfg.Min < 1 {
		return fmt.Errorf("teaming: --min must be >= 1, got %d", cfg.Min)
	}

	if cfg.Max < cfg.Min {
		return fmt.Errorf("teaming: --max (%d) must be >= --min (%d)", cfg.Max, cfg.Min)
	}

	return nil
}
