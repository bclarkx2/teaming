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
// inputs. It runs in three phases:
//
//  1. First-fit-decreasing packing into bins of capacity Max.
//  2. A leftover-merge pass that lifts under-Min teams toward Min by merging
//     under-Min bins WITH EACH OTHER (never exceeding Max).
//  3. A redistribution/repair pass that lifts any remaining under-Min bins
//     toward Min by importing whole groups from healthy donor bins that can
//     spare them (donor stays >= Min, recipient stays <= Max).
//
// The merge and redistribution passes are run in a loop because one can newly
// enable the other; the loop terminates because each pass strictly reduces the
// number of bins (merge) or strictly increases an under-Min bin's size toward
// Min by importing groups bounded by the donor inventory (redistribution).
//
// Neither pass can ever create an over-Max team or drop a donor below Min, so
// they can only hold overMax equal and weakly improve atLeastMin/belowMin —
// they can never worsen the lexicographic score. Every pass is fully
// deterministic (no reliance on map iteration order). Returns teamOfGroup
// indexed by ORIGINAL group index.
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

	// firstGroup returns the smallest original group index in a bin, used as a
	// deterministic tie-break key. Bins always hold at least one group here.
	firstGroup := func(b *bin) int {
		fg := b.groups[0]
		for _, gi := range b.groups[1:] {
			if gi < fg {
				fg = gi
			}
		}
		return fg
	}

	// mergePass combines under-Min bins to lift them toward Min. Repeatedly
	// merge under-Min bins (smallest first) while doing so keeps the result
	// <= Max. Rule 2 (over-Max) outranks rule 4 (below-Min), so we must NOT
	// create an over-Max team to fix a below-Min team; we only merge when the
	// result stays <= Max. Returns true if any merge occurred.
	mergePass := func() bool {
		changed := false
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
						changed = true
						break
					}
				}
			}
			if !merged {
				break
			}
		}
		return changed
	}

	// redistributePass lifts under-Min bins toward Min by importing whole
	// groups from healthy donor bins that can spare them. For each under-Min
	// recipient U (deterministic order: ascending size, then ascending
	// smallest-group-index), it repeatedly searches all other bins D and each
	// group g in D for the largest group that fits without pushing the donor
	// below Min (D.size - sizes[g] >= min) or the recipient over Max
	// (U.size + sizes[g] <= max). Ties break by smallest donor first-group-index
	// then smallest group index. This can only weakly improve the score.
	// Returns true if any move occurred.
	redistributePass := func() bool {
		changed := false
		for {
			// Collect under-Min recipients in deterministic order.
			var underIdx []int
			for i, b := range bins {
				if b.size < min {
					underIdx = append(underIdx, i)
				}
			}
			if len(underIdx) == 0 {
				break
			}
			sort.SliceStable(underIdx, func(a, b int) bool {
				ba, bb := bins[underIdx[a]], bins[underIdx[b]]
				if ba.size != bb.size {
					return ba.size < bb.size
				}
				return firstGroup(ba) < firstGroup(bb)
			})

			movedThisSweep := false
			for _, ui := range underIdx {
				u := bins[ui]
				for u.size < min {
					// Search for the best legal move into U.
					bestDonor := -1
					bestGroupPos := -1
					bestSize := 0
					bestDonorFG := 0
					bestGroupIdx := 0
					for di, d := range bins {
						if di == ui {
							continue
						}
						dfg := firstGroup(d)
						for gp, gi := range d.groups {
							sz := sizes[gi]
							// Guards: donor stays >= Min, recipient stays <= Max.
							if d.size-sz < min {
								continue
							}
							if u.size+sz > max {
								continue
							}
							better := false
							if bestDonor == -1 {
								better = true
							} else if sz != bestSize {
								// Prefer the largest fitting group (fewest moves).
								better = sz > bestSize
							} else if dfg != bestDonorFG {
								better = dfg < bestDonorFG
							} else {
								better = gi < bestGroupIdx
							}
							if better {
								bestDonor = di
								bestGroupPos = gp
								bestSize = sz
								bestDonorFG = dfg
								bestGroupIdx = gi
							}
						}
					}
					if bestDonor == -1 {
						// No legal move can help U; it is unfixable here.
						break
					}
					// Execute the move: remove group from donor, add to U.
					d := bins[bestDonor]
					gi := d.groups[bestGroupPos]
					d.groups = append(d.groups[:bestGroupPos], d.groups[bestGroupPos+1:]...)
					d.size -= bestSize
					u.groups = append(u.groups, gi)
					u.size += bestSize
					changed = true
					movedThisSweep = true
				}
			}

			// Defensively drop any now-empty donor bins so they never become
			// phantom teams. (The donor-stays-valid guard makes this impossible
			// when min >= 1, but handle it anyway.)
			kept := bins[:0]
			for _, b := range bins {
				if len(b.groups) > 0 {
					kept = append(kept, b)
				}
			}
			bins = kept

			if !movedThisSweep {
				break
			}
		}
		return changed
	}

	// Run the merge and redistribution passes in a loop: each can newly enable
	// the other. The loop terminates because every iteration that continues
	// made at least one change, and changes monotonically reduce bin count or
	// raise under-Min bins toward Min from a finite donor inventory.
	for {
		c1 := mergePass()
		c2 := redistributePass()
		if !c1 && !c2 {
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
