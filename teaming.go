// Package teaming assigns whole groups of people to larger teams under a
// strict lexicographic objective.
package teaming

import (
	"fmt"
	"sort"
)

// Person is a single individual belonging to a named group.
type Person struct {
	Name  string
	Group string
}

// Group aggregates the people sharing one group label.
type Group struct {
	Name    string
	Members []string // person names, in input (first-seen) order
}

// Options configures the assignment.
type Options struct {
	Min, Max       int // validated: 1 <= Min <= Max
	ExactThreshold int // max #groups for exact branch-and-bound; <= 0 => DefaultExactThreshold
}

// DefaultExactThreshold is used when Options.ExactThreshold <= 0.
const DefaultExactThreshold = 12

// Assignment is the result for one person; Team is 1-indexed.
type Assignment struct {
	Person string
	Group  string
	Team   int
}

// GroupPeople aggregates people into Groups, preserving first-seen order
// of both groups and members.
func GroupPeople(people []Person) []Group {
	groups := make([]Group, 0)
	index := make(map[string]int)
	for _, p := range people {
		i, ok := index[p.Group]
		if !ok {
			i = len(groups)
			index[p.Group] = i
			groups = append(groups, Group{Name: p.Group})
		}
		groups[i].Members = append(groups[i].Members, p.Name)
	}
	return groups
}

// partitionScore captures the three objective counts derived from a candidate
// partition's team sizes. The objective is a strict lexicographic maximization:
//
//  1. (hard) whole groups stay together — guaranteed by construction.
//  2. minimize overMax    (teams with size > Max)
//  3. maximize atLeastMin (teams with size >= Min)
//  4. minimize belowMin   (teams with size < Min)
//
// Note: every team is exactly one of {size>Max, Min<=size<=Max, size<Min},
// so the three counts fully and consistently describe the objective.
type partitionScore struct {
	overMax    int
	atLeastMin int
	belowMin   int
}

// better reports whether score a is strictly better than score b under the
// lexicographic order described on partitionScore.
func (a partitionScore) better(b partitionScore) bool {
	if a.overMax != b.overMax {
		return a.overMax < b.overMax // rule 2: fewer over-Max is better
	}
	if a.atLeastMin != b.atLeastMin {
		return a.atLeastMin > b.atLeastMin // rule 3: more at-least-Min is better
	}
	return a.belowMin < b.belowMin // rule 4: fewer below-Min is better
}

// scoreSizes computes the partitionScore for a slice of team sizes.
func scoreSizes(sizes []int, min, max int) partitionScore {
	var s partitionScore
	for _, sz := range sizes {
		if sz > max {
			s.overMax++
		}
		if sz >= min {
			s.atLeastMin++
		} else {
			s.belowMin++
		}
	}
	return s
}

// Assign partitions groups into teams under the lexicographic objective.
// Exact branch-and-bound when the number of groups <= effective threshold
// (Options.ExactThreshold if > 0, else DefaultExactThreshold); a greedy
// first-fit-decreasing heuristic otherwise. Returns one Assignment per input
// person, in input order. Team numbers are 1-indexed and assigned stably
// (by ascending order of the first input appearance of each team's groups),
// so output is deterministic. Returns an error only for invalid Options
// (e.g. Min < 1 or Max < Min). Empty input returns an empty slice, no error.
func Assign(people []Person, opts Options) ([]Assignment, error) {
	if opts.Min < 1 {
		return nil, fmt.Errorf("teaming: invalid Options: Min must be >= 1, got %d", opts.Min)
	}
	if opts.Max < opts.Min {
		return nil, fmt.Errorf("teaming: invalid Options: Max (%d) must be >= Min (%d)", opts.Max, opts.Min)
	}

	if len(people) == 0 {
		return []Assignment{}, nil
	}

	threshold := opts.ExactThreshold
	if threshold <= 0 {
		threshold = DefaultExactThreshold
	}

	groups := GroupPeople(people)

	// groupOrder[i] is the first-seen input index of group i (groups are
	// already in first-seen order, so it is just i). We carry sizes plus an
	// ordering key so we can restore stable output ordering after solving on
	// size-sorted groups.
	sizes := make([]int, len(groups))
	for i, g := range groups {
		sizes[i] = len(g.Members)
	}

	// teamOfGroup[i] = team index (in solver space) assigned to group i.
	var teamOfGroup []int
	if len(groups) <= threshold {
		teamOfGroup = solveExact(sizes, opts.Min, opts.Max)
	} else {
		teamOfGroup = solveGreedy(sizes, opts.Min, opts.Max)
	}

	return buildAssignments(people, groups, teamOfGroup), nil
}

// buildAssignments converts a solver-space team-of-group mapping into per-person
// Assignments with stable, 1-indexed team numbers. Teams are numbered by the
// ascending first-input-appearance of their member groups (groups are in
// first-seen order, so by the smallest group index in each team).
func buildAssignments(people []Person, groups []Group, teamOfGroup []int) []Assignment {
	// For each solver team, record the smallest group index it contains.
	firstGroupOfTeam := make(map[int]int)
	for gi, team := range teamOfGroup {
		if cur, ok := firstGroupOfTeam[team]; !ok || gi < cur {
			firstGroupOfTeam[team] = gi
		}
	}

	// Order solver-team ids by their first group index, then assign 1-indexed
	// stable team numbers.
	type teamKey struct {
		team       int
		firstGroup int
	}
	keys := make([]teamKey, 0, len(firstGroupOfTeam))
	for team, fg := range firstGroupOfTeam {
		keys = append(keys, teamKey{team: team, firstGroup: fg})
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].firstGroup < keys[j].firstGroup
	})
	teamNumber := make(map[int]int, len(keys))
	for n, k := range keys {
		teamNumber[k.team] = n + 1
	}

	// Map group name -> 1-indexed team number.
	groupTeam := make(map[string]int, len(groups))
	for gi, g := range groups {
		groupTeam[g.Name] = teamNumber[teamOfGroup[gi]]
	}

	out := make([]Assignment, 0, len(people))
	for _, p := range people {
		out = append(out, Assignment{
			Person: p.Name,
			Group:  p.Group,
			Team:   groupTeam[p.Group],
		})
	}
	return out
}

// solveExact finds the optimal partition of groups (given their sizes) via
// branch-and-bound over set partitions. Groups are processed in descending
// size order; each group is placed into an existing team or a new one.
// Returns teamOfGroup indexed by ORIGINAL group index.
func solveExact(sizes []int, min, max int) []int {
	n := len(sizes)

	// Sort group indices by descending size (stable: ties keep original order).
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return sizes[order[a]] > sizes[order[b]]
	})

	// teamSizes[t] is the current size of team t; assign[k] is the team chosen
	// for order[k]. We track the running best.
	teamSizes := make([]int, 0, n)
	assign := make([]int, n) // indexed by position k in `order`
	bestAssign := make([]int, n)
	var best partitionScore
	haveBest := false

	var recurse func(k int)
	recurse = func(k int) {
		if k == n {
			score := scoreSizes(teamSizes, min, max)
			if !haveBest || score.better(best) {
				best = score
				haveBest = true
				copy(bestAssign, assign)
			}
			return
		}

		// Optimistic bound: even if all remaining groups were placed perfectly,
		// the already-finalized over-Max teams cannot be undone, and the lower
		// bound on over-Max is the current count among full teams. We use a
		// light prune: compute the partial score over current teams and the
		// best achievable from here. A simple admissible prune: the number of
		// over-Max teams only grows or stays as we add groups to existing teams
		// or open new ones, so the current overMax is a lower bound. If
		// haveBest and current overMax already exceeds best.overMax, no
		// completion can beat best on rule 2.
		if haveBest {
			cur := scoreSizes(teamSizes, min, max)
			if cur.overMax > best.overMax {
				return
			}
		}

		gi := order[k]
		sz := sizes[gi]

		// Try placing into each existing team. To avoid exploring symmetric
		// permutations of identical empty teams, we only ever open ONE new team
		// per level (handled below).
		for t := range teamSizes {
			teamSizes[t] += sz
			assign[k] = t
			recurse(k + 1)
			teamSizes[t] -= sz
		}

		// Open a new team for this group.
		teamSizes = append(teamSizes, sz)
		assign[k] = len(teamSizes) - 1
		recurse(k + 1)
		teamSizes = teamSizes[:len(teamSizes)-1]
	}

	recurse(0)

	// Translate bestAssign (positions in `order`, solver team ids) back to
	// original group indices.
	teamOfGroup := make([]int, n)
	for k, gi := range order {
		teamOfGroup[gi] = bestAssign[k]
	}
	return teamOfGroup
}

// solveGreedy is a near-optimal (not guaranteed optimal) heuristic for large
// inputs: first-fit-decreasing packing into bins of capacity Max, followed by
// a leftover-merge pass that lifts under-Min teams toward Min by merging them.
// Returns teamOfGroup indexed by ORIGINAL group index.
func solveGreedy(sizes []int, min, max int) []int {
	n := len(sizes)

	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return sizes[order[a]] > sizes[order[b]]
	})

	// Each bin holds the original group indices placed in it, plus a size.
	type bin struct {
		groups []int
		size   int
	}
	var bins []*bin

	// First-fit-decreasing: place each group into the first bin that keeps it
	// within Max; otherwise open a new bin. A group larger than Max gets its
	// own bin (which becomes an over-Max team — never an error).
	for _, gi := range order {
		sz := sizes[gi]
		placed := false
		for _, b := range bins {
			if b.size+sz <= max {
				b.groups = append(b.groups, gi)
				b.size += sz
				placed = true
				break
			}
		}
		if !placed {
			bins = append(bins, &bin{groups: []int{gi}, size: sz})
		}
	}

	// Leftover-merge pass: combine under-Min bins to lift them toward Min.
	// Repeatedly merge the two smallest under-Min bins while doing so does not
	// create an over-Max team (prefer staying within Max). If merging two
	// under-Min bins would exceed Max we still merge them only when both are
	// under Min and no Max-respecting merge is available, since reducing
	// below-Min count is a higher objective priority than respecting Max... but
	// rule 2 (over-Max) outranks rule 4 (below-Min), so we must NOT create an
	// over-Max team to fix a below-Min team. Therefore we only merge when the
	// result stays <= Max.
	for {
		// Collect under-Min bins.
		var underIdx []int
		for i, b := range bins {
			if b.size < min {
				underIdx = append(underIdx, i)
			}
		}
		if len(underIdx) < 2 {
			break
		}
		// Sort under-Min bins by ascending size for a stable, greedy merge.
		sort.SliceStable(underIdx, func(a, b int) bool {
			return bins[underIdx[a]].size < bins[underIdx[b]].size
		})

		// Find a pair of under-Min bins whose merged size stays <= Max.
		merged := false
		for x := 0; x < len(underIdx) && !merged; x++ {
			for y := x + 1; y < len(underIdx); y++ {
				i, j := underIdx[x], underIdx[y]
				if bins[i].size+bins[j].size <= max {
					bins[i].groups = append(bins[i].groups, bins[j].groups...)
					bins[i].size += bins[j].size
					// Remove bin j.
					bins = append(bins[:j], bins[j+1:]...)
					merged = true
					break
				}
			}
		}
		if !merged {
			break
		}
	}

	teamOfGroup := make([]int, n)
	for t, b := range bins {
		for _, gi := range b.groups {
			teamOfGroup[gi] = t
		}
	}
	return teamOfGroup
}
