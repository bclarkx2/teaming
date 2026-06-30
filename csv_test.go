package teaming

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadPeople(t *testing.T) {
	run := func(name, input string, wantPeople []Person, wantErr string) func(*testing.T) {
		return func(t *testing.T) {
			t.Helper()
			got, err := ReadPeople(strings.NewReader(input))
			if wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", wantErr)
				}
				if !strings.Contains(err.Error(), wantErr) {
					t.Fatalf("expected error containing %q, got: %v", wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(wantPeople) {
				t.Fatalf("got %d people, want %d: %v", len(got), len(wantPeople), got)
			}
			for i, p := range got {
				if p != wantPeople[i] {
					t.Errorf("row %d: got %+v, want %+v", i, p, wantPeople[i])
				}
			}
		}
	}

	t.Run("basic", run(
		"basic",
		"person,group\nAlice,A\nBob,B\n",
		[]Person{{Name: "Alice", Group: "A"}, {Name: "Bob", Group: "B"}},
		"",
	))

	t.Run("column_order_independence", run(
		"column_order_independence",
		"group,person\nA,Alice\nB,Bob\n",
		[]Person{{Name: "Alice", Group: "A"}, {Name: "Bob", Group: "B"}},
		"",
	))

	t.Run("case_insensitive_headers", run(
		"case_insensitive_headers",
		"Person,Group\nAlice,A\nBob,B\n",
		[]Person{{Name: "Alice", Group: "A"}, {Name: "Bob", Group: "B"}},
		"",
	))

	t.Run("ignores_team_column", run(
		"ignores_team_column",
		"person,group,team\nAlice,A,1\nBob,B,2\n",
		[]Person{{Name: "Alice", Group: "A"}, {Name: "Bob", Group: "B"}},
		"",
	))

	t.Run("ignores_extra_columns", run(
		"ignores_extra_columns",
		"person,group,team,notes\nAlice,A,1,hello\nBob,B,2,world\n",
		[]Person{{Name: "Alice", Group: "A"}, {Name: "Bob", Group: "B"}},
		"",
	))

	t.Run("missing_person_column", run(
		"missing_person_column",
		"name,group\nAlice,A\n",
		nil,
		`required column "person" not found`,
	))

	t.Run("missing_group_column", run(
		"missing_group_column",
		"person,department\nAlice,eng\n",
		nil,
		`required column "group" not found`,
	))

	t.Run("empty_input", run(
		"empty_input",
		"",
		nil,
		"empty input",
	))

	t.Run("mixed_case_headers", run(
		"mixed_case_headers",
		"PERSON,GROUP\nAlice,A\n",
		[]Person{{Name: "Alice", Group: "A"}},
		"",
	))
}

func TestWriteAssignments(t *testing.T) {
	run := func(name string, assignments []Assignment, wantLines []string, wantErr string) func(*testing.T) {
		return func(t *testing.T) {
			t.Helper()
			var buf bytes.Buffer
			err := WriteAssignments(&buf, assignments)
			if wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", wantErr)
				}
				if !strings.Contains(err.Error(), wantErr) {
					t.Fatalf("expected error containing %q, got: %v", wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
			if len(got) != len(wantLines) {
				t.Fatalf("got %d lines, want %d:\n%s", len(got), len(wantLines), buf.String())
			}
			for i, line := range got {
				if line != wantLines[i] {
					t.Errorf("line %d: got %q, want %q", i, line, wantLines[i])
				}
			}
		}
	}

	t.Run("basic", run(
		"basic",
		[]Assignment{
			{Person: "Alice", Group: "A", Team: 1},
			{Person: "Bob", Group: "B", Team: 2},
		},
		[]string{"person,group,team", "Alice,A,1", "Bob,B,2"},
		"",
	))

	t.Run("empty_assignments", run(
		"empty_assignments",
		[]Assignment{},
		[]string{"person,group,team"},
		"",
	))

	t.Run("multiple_teams", run(
		"multiple_teams",
		[]Assignment{
			{Person: "Alice", Group: "A", Team: 1},
			{Person: "Bob", Group: "A", Team: 1},
			{Person: "Carol", Group: "B", Team: 2},
		},
		[]string{"person,group,team", "Alice,A,1", "Bob,A,1", "Carol,B,2"},
		"",
	))
}

func TestRoundTrip(t *testing.T) {
	run := func(name string, people []Person, opts Options) func(*testing.T) {
		return func(t *testing.T) {
			t.Helper()

			assignments, err := Assign(people, opts)
			if err != nil {
				t.Fatalf("Assign: %v", err)
			}

			// Write assignments to a buffer.
			var buf bytes.Buffer
			if err := WriteAssignments(&buf, assignments); err != nil {
				t.Fatalf("WriteAssignments: %v", err)
			}

			// Read the written CSV back as people (ignoring team column).
			got, err := ReadPeople(&buf)
			if err != nil {
				t.Fatalf("ReadPeople: %v", err)
			}

			if len(got) != len(people) {
				t.Fatalf("round-trip: got %d people, want %d", len(got), len(people))
			}

			// Verify names and groups are preserved; team column is silently ignored.
			for i, p := range got {
				if p.Name != people[i].Name {
					t.Errorf("row %d: Name got %q, want %q", i, p.Name, people[i].Name)
				}
				if p.Group != people[i].Group {
					t.Errorf("row %d: Group got %q, want %q", i, p.Group, people[i].Group)
				}
			}
		}
	}

	t.Run("two_teams", run(
		"two_teams",
		[]Person{
			{Name: "Alice", Group: "A"},
			{Name: "Bob", Group: "A"},
			{Name: "Carol", Group: "B"},
			{Name: "Dave", Group: "B"},
		},
		Options{Min: 2, Max: 4},
	))

	t.Run("single_team", run(
		"single_team",
		[]Person{
			{Name: "Alice", Group: "A"},
			{Name: "Bob", Group: "B"},
		},
		Options{Min: 1, Max: 5},
	))
}
