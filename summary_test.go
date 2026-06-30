package teaming

import (
	"testing"
)

func TestSummarize(t *testing.T) {
	type tc struct {
		name        string
		assignments []Assignment
		want        []TeamSummary
	}
	run := func(c tc) func(*testing.T) {
		return func(t *testing.T) {
			got := Summarize(c.assignments)
			if len(got) != len(c.want) {
				t.Fatalf("summary count = %d, want %d (got %v)", len(got), len(c.want), got)
			}
			for i := range got {
				if got[i].Team != c.want[i].Team {
					t.Fatalf("summary[%d].Team = %d, want %d", i, got[i].Team, c.want[i].Team)
				}
				if got[i].People != c.want[i].People {
					t.Fatalf("summary[%d].People = %d, want %d", i, got[i].People, c.want[i].People)
				}
				if got[i].Groups != c.want[i].Groups {
					t.Fatalf("summary[%d].Groups = %d, want %d", i, got[i].Groups, c.want[i].Groups)
				}
			}
		}
	}
	for _, c := range []tc{
		{
			name:        "empty input",
			assignments: nil,
			want:        []TeamSummary{},
		},
		{
			name: "single team",
			assignments: []Assignment{
				{Person: "alice", Group: "A", Team: 1},
				{Person: "bob", Group: "A", Team: 1},
				{Person: "carol", Group: "B", Team: 1},
			},
			want: []TeamSummary{
				{Team: 1, People: 3, Groups: 2},
			},
		},
		{
			name: "multi-team with repeated groups",
			assignments: []Assignment{
				// Team 1: group A (3 people) + group B (2 people) = 5 people, 2 groups
				{Person: "alice", Group: "A", Team: 1},
				{Person: "bob", Group: "A", Team: 1},
				{Person: "carol", Group: "A", Team: 1},
				{Person: "dave", Group: "B", Team: 1},
				{Person: "eve", Group: "B", Team: 1},
				// Team 2: group C (1 person) + group D (3 people) = 4 people, 2 groups
				{Person: "frank", Group: "C", Team: 2},
				{Person: "grace", Group: "D", Team: 2},
				{Person: "hank", Group: "D", Team: 2},
				{Person: "iris", Group: "D", Team: 2},
			},
			want: []TeamSummary{
				{Team: 1, People: 5, Groups: 2},
				{Team: 2, People: 4, Groups: 2},
			},
		},
		{
			name: "ordering by team number ascending",
			assignments: []Assignment{
				// Provide in reverse team order to verify sorting.
				{Person: "zara", Group: "Z", Team: 3},
				{Person: "mike", Group: "M", Team: 2},
				{Person: "mike2", Group: "M", Team: 2},
				{Person: "alpha", Group: "A", Team: 1},
				{Person: "beta", Group: "A", Team: 1},
				{Person: "gamma", Group: "B", Team: 1},
			},
			want: []TeamSummary{
				{Team: 1, People: 3, Groups: 2},
				{Team: 2, People: 2, Groups: 1},
				{Team: 3, People: 1, Groups: 1},
			},
		},
		{
			name: "group contributing many people counts as 1 group",
			assignments: []Assignment{
				{Person: "p1", Group: "Big", Team: 1},
				{Person: "p2", Group: "Big", Team: 1},
				{Person: "p3", Group: "Big", Team: 1},
				{Person: "p4", Group: "Big", Team: 1},
				{Person: "p5", Group: "Big", Team: 1},
			},
			want: []TeamSummary{
				{Team: 1, People: 5, Groups: 1},
			},
		},
	} {
		t.Run(c.name, run(c))
	}
}
