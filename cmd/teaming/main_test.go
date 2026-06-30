package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newTestViper returns a freshly-initialised viper instance suitable for
// isolated tests (no shared global state).
func newTestViper() *viper.Viper {
	return viper.New()
}

// applyFlagsToViper simulates what bindFlags + Resolve do, but operates on a
// controlled viper instance populated directly — no cobra flag parsing needed.
func resolveFromMap(m map[string]interface{}) Config {
	v := newTestViper()
	for k, val := range m {
		v.Set(k, val)
	}
	return Resolve(v)
}

// TestValidate checks all Validate() paths.
func TestValidate(t *testing.T) {
	run := func(name string, cfg Config, wantErr string) func(*testing.T) {
		return func(t *testing.T) {
			t.Helper()
			err := cfg.Validate()
			if wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", wantErr)
				}
				if !contains(err.Error(), wantErr) {
					t.Fatalf("expected error containing %q, got: %v", wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
	}

	t.Run("valid", run(
		"valid",
		Config{Input: "in.csv", Output: "out.csv", Min: 3, Max: 5},
		"",
	))

	t.Run("missing_input", run(
		"missing_input",
		Config{Output: "out.csv", Min: 3, Max: 5},
		"--input is required",
	))

	t.Run("min_zero", run(
		"min_zero",
		Config{Input: "in.csv", Min: 0, Max: 5},
		"--min must be >= 1",
	))

	t.Run("min_negative", run(
		"min_negative",
		Config{Input: "in.csv", Min: -1, Max: 5},
		"--min must be >= 1",
	))

	t.Run("max_less_than_min", run(
		"max_less_than_min",
		Config{Input: "in.csv", Min: 5, Max: 3},
		"--max (3) must be >= --min (5)",
	))

	t.Run("output_defaults_to_input", run(
		"output_defaults_to_input",
		Config{Input: "in.csv", Min: 2, Max: 4},
		"",
	))

	t.Run("exact_threshold_negative", run(
		"exact_threshold_negative",
		Config{Input: "in.csv", Output: "out.csv", Min: 3, Max: 5, ExactThreshold: -1},
		"--exact-threshold must be >= 0",
	))

	t.Run("exact_threshold_zero_ok", run(
		"exact_threshold_zero_ok",
		Config{Input: "in.csv", Output: "out.csv", Min: 3, Max: 5, ExactThreshold: 0},
		"",
	))

	t.Run("exact_threshold_positive_ok", run(
		"exact_threshold_positive_ok",
		Config{Input: "in.csv", Output: "out.csv", Min: 3, Max: 5, ExactThreshold: 20},
		"",
	))
}

// TestValidateOutputDefaultsToInput verifies the output-defaults-to-input behaviour.
func TestValidateOutputDefaultsToInput(t *testing.T) {
	cfg := Config{Input: "people.csv", Min: 2, Max: 4}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Output != "people.csv" {
		t.Errorf("output should default to input, got %q", cfg.Output)
	}
}

// TestConfigPrecedence verifies that viper key setting respects explicit Set
// (simulating flag or env) over lower-priority sources.
func TestConfigPrecedence(t *testing.T) {
	run := func(name string, setup func(*viper.Viper), want Config) func(*testing.T) {
		return func(t *testing.T) {
			t.Helper()
			v := newTestViper()
			setup(v)
			got := Resolve(v)
			if got.Input != want.Input {
				t.Errorf("Input: got %q, want %q", got.Input, want.Input)
			}
			if got.Output != want.Output {
				t.Errorf("Output: got %q, want %q", got.Output, want.Output)
			}
			if got.Min != want.Min {
				t.Errorf("Min: got %d, want %d", got.Min, want.Min)
			}
			if got.Max != want.Max {
				t.Errorf("Max: got %d, want %d", got.Max, want.Max)
			}
			if got.ExactThreshold != want.ExactThreshold {
				t.Errorf("ExactThreshold: got %d, want %d", got.ExactThreshold, want.ExactThreshold)
			}
		}
	}

	t.Run("explicit_values", run(
		"explicit_values",
		func(v *viper.Viper) {
			v.Set("input", "a.csv")
			v.Set("output", "b.csv")
			v.Set("min", 3)
			v.Set("max", 6)
		},
		Config{Input: "a.csv", Output: "b.csv", Min: 3, Max: 6},
	))

	t.Run("env_vars", run(
		"env_vars",
		func(v *viper.Viper) {
			v.SetEnvPrefix("TEAMING")
			v.AutomaticEnv()
			t.Setenv("TEAMING_INPUT", "env_in.csv")
			t.Setenv("TEAMING_OUTPUT", "env_out.csv")
			t.Setenv("TEAMING_MIN", "2")
			t.Setenv("TEAMING_MAX", "7")
		},
		Config{Input: "env_in.csv", Output: "env_out.csv", Min: 2, Max: 7},
	))

	t.Run("flag_beats_env", run(
		"flag_beats_env",
		func(v *viper.Viper) {
			v.SetEnvPrefix("TEAMING")
			v.AutomaticEnv()
			t.Setenv("TEAMING_INPUT", "env_in.csv")
			t.Setenv("TEAMING_MIN", "2")
			t.Setenv("TEAMING_MAX", "7")
			// Simulate flag override (viper.Set has highest priority).
			v.Set("input", "flag_in.csv")
			v.Set("min", 4)
		},
		Config{Input: "flag_in.csv", Output: "env_out.csv", Min: 4, Max: 7},
	))

	t.Run("exact_threshold_default_zero", run(
		"exact_threshold_default_zero",
		func(v *viper.Viper) {
			v.Set("input", "a.csv")
			v.Set("output", "b.csv")
			v.Set("min", 3)
			v.Set("max", 6)
			// exact-threshold not set → resolves to zero (library default)
		},
		Config{Input: "a.csv", Output: "b.csv", Min: 3, Max: 6, ExactThreshold: 0},
	))

	// ExactThreshold-specific tests use their own subtest t for isolated env vars.
	t.Run("exact_threshold_env_var", func(t *testing.T) {
		t.Helper()
		t.Setenv("TEAMING_EXACT_THRESHOLD", "20")
		v := newTestViper()
		v.SetEnvPrefix("TEAMING")
		v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		v.AutomaticEnv()
		got := Resolve(v)
		if got.ExactThreshold != 20 {
			t.Errorf("ExactThreshold: got %d, want 20", got.ExactThreshold)
		}
	})

	t.Run("exact_threshold_flag_beats_env", func(t *testing.T) {
		t.Helper()
		t.Setenv("TEAMING_EXACT_THRESHOLD", "20")
		v := newTestViper()
		v.SetEnvPrefix("TEAMING")
		v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		v.AutomaticEnv()
		// Simulate flag override (viper.Set has highest priority).
		v.Set("exact-threshold", 30)
		got := Resolve(v)
		if got.ExactThreshold != 30 {
			t.Errorf("ExactThreshold: got %d, want 30", got.ExactThreshold)
		}
	})

	t.Run("exact_threshold_config_file", func(t *testing.T) {
		t.Helper()
		dir := t.TempDir()
		p := filepath.Join(dir, "teaming.yaml")
		content := "exact-threshold: 15\n"
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("writing test config: %v", err)
		}
		v := newTestViper()
		if err := loadConfigFile(v, p); err != nil {
			t.Fatalf("loading config: %v", err)
		}
		got := Resolve(v)
		if got.ExactThreshold != 15 {
			t.Errorf("ExactThreshold: got %d, want 15", got.ExactThreshold)
		}
	})
}

// TestConfigFileLoading verifies that loadConfigFile reads YAML and that a
// missing file is not an error.
func TestConfigFileLoading(t *testing.T) {
	run := func(name string, writeFile func(dir string) (path string), setKey string, wantVal string, wantErr string) func(*testing.T) {
		return func(t *testing.T) {
			t.Helper()
			dir := t.TempDir()
			v := newTestViper()

			configPath := writeFile(dir)

			if err := loadConfigFile(v, configPath); err != nil {
				if wantErr == "" {
					t.Fatalf("unexpected error: %v", err)
				}
				if !contains(err.Error(), wantErr) {
					t.Fatalf("expected error containing %q, got: %v", wantErr, err)
				}
				return
			}
			if wantErr != "" {
				t.Fatalf("expected error containing %q, got nil", wantErr)
			}

			if setKey != "" {
				got := v.GetString(setKey)
				if got != wantVal {
					t.Errorf("key %q: got %q, want %q", setKey, got, wantVal)
				}
			}
		}
	}

	t.Run("missing_file_no_error", run(
		"missing_file_no_error",
		func(dir string) string {
			return "" // no explicit path, auto-search will find nothing
		},
		"",
		"",
		"",
	))

	t.Run("reads_yaml", run(
		"reads_yaml",
		func(dir string) string {
			p := filepath.Join(dir, "teaming.yaml")
			content := "input: file_in.csv\nmin: 3\nmax: 5\n"
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				panic(fmt.Sprintf("writing test config: %v", err))
			}
			return p
		},
		"input",
		"file_in.csv",
		"",
	))
}

// TestBindFlags verifies that BindPFlag wires cobra flags to viper correctly.
func TestBindFlags(t *testing.T) {
	run := func(name string, flagArgs []string, wantInput string, wantMin int) func(*testing.T) {
		return func(t *testing.T) {
			t.Helper()
			v := newTestViper()

			cmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error {
				return bindFlags(v, cmd)
			}}
			cmd.Flags().StringP("input", "i", "", "input file")
			cmd.Flags().StringP("output", "o", "", "output file")
			cmd.Flags().Int("min", 0, "min team size")
			cmd.Flags().Int("max", 0, "max team size")
			cmd.Flags().Int("exact-threshold", 0, "exact threshold")

			cmd.SetArgs(flagArgs)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute: %v", err)
			}

			if got := v.GetString("input"); got != wantInput {
				t.Errorf("input: got %q, want %q", got, wantInput)
			}
			if got := v.GetInt("min"); got != wantMin {
				t.Errorf("min: got %d, want %d", got, wantMin)
			}
		}
	}

	t.Run("short_flags", run(
		"short_flags",
		[]string{"-i", "people.csv", "--min", "3", "--max", "6"},
		"people.csv",
		3,
	))

	t.Run("long_flags", run(
		"long_flags",
		[]string{"--input", "people.csv", "--min", "5", "--max", "10"},
		"people.csv",
		5,
	))
}

// contains is a simple substring helper to avoid importing strings in the test
// file's header if it would otherwise not be needed.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
