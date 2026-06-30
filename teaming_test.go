package teaming

import (
	"sort"
	"testing"
)

func TestGroupPeople(t *testing.T) {
	type tc struct {
		name   string
		people []Person
		want   []Group
	}
	run := func(c tc) func(*testing.T) {
		return func(t *testing.T) {
			got := GroupPeople(c.people)
			if len(got) != len(c.want) {
				t.Fatalf("group count = %d, want %d (%v)", len(got), len(c.want), got)
			}
			for i := range got {
				if got[i].Name != c.want[i].Name {
					t.Fatalf("group[%d].Name = %q, want %q", i, got[i].Name, c.want[i].Name)
				}
				if len(got[i].Members) != len(c.want[i].Members) {
					t.Fatalf("group[%d] members = %v, want %v", i, got[i].Members, c.want[i].Members)
				}
				for j := range got[i].Members {
					if got[i].Members[j] != c.want[i].Members[j] {
						t.Fatalf("group[%d].Members = %v, want %v", i, got[i].Members, c.want[i].Members)
					}
				}
			}
		}
	}
	for _, c := range []tc{
		{
			name:   "empty",
			people: nil,
			want:   []Group{},
		},
		{
			name: "first-seen order preserved",
			people: []Person{
				{Name: "john", Group: "A"},
				{Name: "jack", Group: "B"},
				{Name: "mary", Group: "A"},
				{Name: "tara", Group: "B"},
			},
			want: []Group{
				{Name: "A", Members: []string{"john", "mary"}},
				{Name: "B", Members: []string{"jack", "tara"}},
			},
		},
		{
			name: "single person single group",
			people: []Person{
				{Name: "solo", Group: "Z"},
			},
			want: []Group{
				{Name: "Z", Members: []string{"solo"}},
			},
		},
	} {
		t.Run(c.name, run(c))
	}
}

// teamSizes returns the multiset of team sizes (sorted ascending) and the
// per-team membership derived from a set of Assignments.
func teamSizes(as []Assignment) []int {
	counts := map[int]int{}
	for _, a := range as {
		counts[a.Team]++
	}
	sizes := make([]int, 0, len(counts))
	for _, c := range counts {
		sizes = append(sizes, c)
	}
	sort.Ints(sizes)
	return sizes
}

// groupTeams maps each group name to the team number it landed in, and fails
// the test if a single group is split across multiple teams.
func groupTeams(t *testing.T, as []Assignment) map[string]int {
	t.Helper()
	m := map[string]int{}
	for _, a := range as {
		if prev, ok := m[a.Group]; ok && prev != a.Team {
			t.Fatalf("group %q split across teams %d and %d", a.Group, prev, a.Team)
		}
		m[a.Group] = a.Team
	}
	return m
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestAssign(t *testing.T) {
	type tc struct {
		name      string
		people    []Person
		opts      Options
		wantErr   bool
		wantSizes []int      // expected team size multiset (sorted asc); nil to skip
		wantLen   int        // expected number of assignments; -1 to skip
		sameTeam  [][]string // each inner slice of group names must share a team
		check     func(*testing.T, []Assignment)
	}
	run := func(c tc) func(*testing.T) {
		return func(t *testing.T) {
			got, err := Assign(c.people, c.opts)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.wantLen >= 0 && len(got) != c.wantLen {
				t.Fatalf("assignment count = %d, want %d", len(got), c.wantLen)
			}
			// Output order must match input order.
			for i := range got {
				if got[i].Person != c.people[i].Name || got[i].Group != c.people[i].Group {
					t.Fatalf("assignment[%d] = %+v, expected person %q group %q",
						i, got[i], c.people[i].Name, c.people[i].Group)
				}
			}
			gt := groupTeams(t, got)
			for _, set := range c.sameTeam {
				for _, g := range set[1:] {
					if gt[g] != gt[set[0]] {
						t.Fatalf("groups %q and %q expected same team, got %d and %d",
							set[0], g, gt[set[0]], gt[g])
					}
				}
			}
			if c.wantSizes != nil {
				if sz := teamSizes(got); !equalInts(sz, c.wantSizes) {
					t.Fatalf("team sizes = %v, want %v", sz, c.wantSizes)
				}
			}
			if c.check != nil {
				c.check(t, got)
			}
		}
	}

	for _, c := range []tc{
		{
			name: "prompt example",
			people: []Person{
				{Name: "john", Group: "A"},
				{Name: "mary", Group: "A"},
				{Name: "jack", Group: "B"},
				{Name: "tara", Group: "B"},
				{Name: "alex", Group: "B"},
				{Name: "mimi", Group: "C"},
				{Name: "naomi", Group: "D"},
			},
			opts:      Options{Min: 3, Max: 4},
			wantLen:   7,
			wantSizes: []int{3, 4},
			sameTeam:  [][]string{{"A"}, {"B"}},
			check: func(t *testing.T, as []Assignment) {
				// Exactly 2 teams.
				if got := len(teamSizes(as)); got != 2 {
					t.Fatalf("expected 2 teams, got %d", got)
				}
			},
		},
		{
			name:    "empty input",
			people:  nil,
			opts:    Options{Min: 2, Max: 4},
			wantLen: 0,
		},
		{
			name:    "invalid Min < 1",
			people:  []Person{{Name: "a", Group: "G"}},
			opts:    Options{Min: 0, Max: 4},
			wantErr: true,
		},
		{
			name:    "invalid Max < Min",
			people:  []Person{{Name: "a", Group: "G"}},
			opts:    Options{Min: 5, Max: 4},
			wantErr: true,
		},
		{
			name: "oversized single group forms its own over-Max team",
			people: []Person{
				{Name: "a", Group: "Big"},
				{Name: "b", Group: "Big"},
				{Name: "c", Group: "Big"},
				{Name: "d", Group: "Big"},
				{Name: "e", Group: "Big"},
			},
			opts:      Options{Min: 2, Max: 3},
			wantLen:   5,
			wantSizes: []int{5},
			sameTeam:  [][]string{{"Big"}},
		},
		{
			name: "leftover below Min unavoidable",
			people: []Person{
				{Name: "a", Group: "A"},
				{Name: "b", Group: "A"},
				{Name: "c", Group: "A"},
				{Name: "d", Group: "B"},
			},
			opts:    Options{Min: 3, Max: 3},
			wantLen: 4,
			// A=3 fills a team; B=1 must be its own under-Min team.
			wantSizes: []int{1, 3},
			sameTeam:  [][]string{{"A"}, {"B"}},
		},
		{
			name: "min equals max exact pack",
			people: func() []Person {
				var ps []Person
				// six singleton groups, Min=Max=2 => three teams of 2.
				names := []string{"a", "b", "c", "d", "e", "f"}
				for _, n := range names {
					ps = append(ps, Person{Name: n, Group: n})
				}
				return ps
			}(),
			opts:      Options{Min: 2, Max: 2},
			wantLen:   6,
			wantSizes: []int{2, 2, 2},
		},
	} {
		t.Run(c.name, run(c))
	}
}

// TestAssignGreedyFallback exercises the greedy path with more groups than
// DefaultExactThreshold and asserts the result is a valid full partition:
// everyone assigned exactly once and every group intact.
func TestAssignGreedyFallback(t *testing.T) {
	type tc struct {
		name   string
		people []Person
		opts   Options
	}
	run := func(c tc) func(*testing.T) {
		return func(t *testing.T) {
			got, err := Assign(c.people, c.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(c.people) {
				t.Fatalf("got %d assignments, want %d", len(got), len(c.people))
			}
			// Each person appears exactly once, in input order.
			for i := range got {
				if got[i].Person != c.people[i].Name || got[i].Group != c.people[i].Group {
					t.Fatalf("assignment[%d] mismatch: %+v", i, got[i])
				}
			}
			// Groups intact: a group never spans two teams.
			groupTeams(t, got)
			// Every assignment has a valid 1-indexed team.
			for _, a := range got {
				if a.Team < 1 {
					t.Fatalf("non-positive team number %d for %q", a.Team, a.Person)
				}
			}
		}
	}

	// Build > DefaultExactThreshold groups (20 groups) of varying sizes.
	var people []Person
	for g := 0; g < 20; g++ {
		groupName := string(rune('A' + g))
		size := (g % 3) + 1 // sizes 1,2,3 cycling
		for m := 0; m < size; m++ {
			people = append(people, Person{
				Name:  groupName + "-" + string(rune('0'+m)),
				Group: groupName,
			})
		}
	}

	for _, c := range []tc{
		{
			name:   "20 groups greedy path",
			people: people,
			opts:   Options{Min: 3, Max: 5, ExactThreshold: 12},
		},
	} {
		t.Run(c.name, run(c))
	}
}

// TestAssignRedistributionRepair reproduces the failure class where FFD packs
// groups into several full Max-capacity teams plus one small leftover team
// below Min, and the leftover-merge pass cannot help (only one under-Min bin).
// The redistribution/repair pass must lift the leftover to >= Min by importing
// small whole groups from healthy donor teams. Mirrors the party numbers:
// Min=4, Max=6, 25 groups summing to 45 people, with many size-2 and several
// size-1 groups available to donate.
func TestAssignRedistributionRepair(t *testing.T) {
	// 5 groups of size 3 (15) + 10 groups of size 2 (20) + 10 groups of
	// size 1 (10) = 45 people across 25 groups.
	var people []Person
	gid := 0
	add := func(size, count int) {
		for c := 0; c < count; c++ {
			groupName := "G" + string(rune('A'+gid))
			gid++
			for m := 0; m < size; m++ {
				people = append(people, Person{
					Name:  groupName + "-" + string(rune('0'+m)),
					Group: groupName,
				})
			}
		}
	}
	add(3, 5)
	add(2, 10)
	add(1, 10)

	opts := Options{Min: 4, Max: 6, ExactThreshold: 5} // force greedy (25 > 5)
	got, err := Assign(people, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Every person assigned exactly once, in input order.
	if len(got) != len(people) {
		t.Fatalf("got %d assignments, want %d", len(got), len(people))
	}
	for i := range got {
		if got[i].Person != people[i].Name || got[i].Group != people[i].Group {
			t.Fatalf("assignment[%d] mismatch: %+v", i, got[i])
		}
	}

	// Groups intact: a group never spans two teams.
	groupTeams(t, got)

	// No team over Max, and zero teams below Min.
	sizes := teamSizes(got)
	for _, sz := range sizes {
		if sz > opts.Max {
			t.Fatalf("team of size %d exceeds Max=%d (sizes=%v)", sz, opts.Max, sizes)
		}
		if sz < opts.Min {
			t.Fatalf("team of size %d below Min=%d (sizes=%v)", sz, opts.Min, sizes)
		}
	}

	// Sanity: all 45 people accounted for.
	total := 0
	for _, sz := range sizes {
		total += sz
	}
	if total != len(people) {
		t.Fatalf("team sizes sum to %d, want %d", total, len(people))
	}
}

// TestAssignDeterminism verifies byte-identical output across repeated runs.
func TestAssignDeterminism(t *testing.T) {
	people := []Person{
		{Name: "john", Group: "A"},
		{Name: "mary", Group: "A"},
		{Name: "jack", Group: "B"},
		{Name: "tara", Group: "B"},
		{Name: "alex", Group: "B"},
		{Name: "mimi", Group: "C"},
		{Name: "naomi", Group: "D"},
	}
	opts := Options{Min: 3, Max: 4}
	first, err := Assign(people, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 5; i++ {
		got, err := Assign(people, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != len(first) {
			t.Fatalf("run %d length differs", i)
		}
		for j := range got {
			if got[j] != first[j] {
				t.Fatalf("run %d differs at %d: %+v vs %+v", i, j, got[j], first[j])
			}
		}
	}
}
