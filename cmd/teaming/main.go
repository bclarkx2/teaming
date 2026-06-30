package main

import (
	"fmt"
	"os"

	"github.com/bclarkx2/teaming"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRootCmd() *cobra.Command {
	v := viper.New()

	cmd := &cobra.Command{
		Use:   "teaming",
		Short: "Assign groups of people to balanced teams",
		Long: `teaming reads a CSV of people and their groups, then assigns whole groups
to teams while respecting the given minimum and maximum team sizes.

The input CSV must have at least a "person" and "group" column (case-insensitive).
Any existing "team" column is ignored on read, making in-place regeneration safe.

When --output is omitted it defaults to --input (in-place update).`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("config")

			if err := bindFlags(v, cmd); err != nil {
				return err
			}

			if err := loadConfigFile(v, configFile); err != nil {
				return err
			}

			cfg := Resolve(v)
			if err := cfg.Validate(); err != nil {
				return err
			}

			// --- Read input (fully, then close) before opening output. ---
			// This is essential when input == output (in-place regeneration).
			inFile, err := os.Open(cfg.Input)
			if err != nil {
				return fmt.Errorf("teaming: opening input %q: %w", cfg.Input, err)
			}

			people, err := teaming.ReadPeople(inFile)
			if err != nil {
				inFile.Close()
				return fmt.Errorf("teaming: reading input: %w", err)
			}
			inFile.Close()

			// --- Assign ---
			opts := teaming.Options{
				Min:            cfg.Min,
				Max:            cfg.Max,
				ExactThreshold: cfg.ExactThreshold,
			}

			assignments, err := teaming.Assign(people, opts)
			if err != nil {
				return fmt.Errorf("teaming: assigning: %w", err)
			}

			// Count distinct teams for the summary.
			teamSet := make(map[int]struct{})
			for _, a := range assignments {
				teamSet[a.Team] = struct{}{}
			}

			// --- Write output (truncate) ---
			outFile, err := os.OpenFile(cfg.Output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
			if err != nil {
				return fmt.Errorf("teaming: opening output %q: %w", cfg.Output, err)
			}
			defer outFile.Close()

			if err := teaming.WriteAssignments(outFile, assignments); err != nil {
				return fmt.Errorf("teaming: writing output: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Assigned %d people across %d teams → %s\n",
				len(people), len(teamSet), cfg.Output)

			return nil
		},
	}

	// Flags — bound to viper in bindFlags (called inside RunE so we have the cobra.Command).
	cmd.Flags().StringP("input", "i", "", "input CSV file (required)")
	cmd.Flags().StringP("output", "o", "", "output CSV file (defaults to --input for in-place update)")
	cmd.Flags().Int("min", 0, "minimum team size (required, >= 1)")
	cmd.Flags().Int("max", 0, "maximum team size (required, >= min)")
	cmd.Flags().Int("exact-threshold", 0, "Maximum number of groups for the exact optimal solver; above this the faster greedy heuristic is used. 0 uses the built-in default (12).")
	cmd.Flags().String("config", "", "path to YAML config file (default: teaming.yaml in working directory)")

	return cmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
