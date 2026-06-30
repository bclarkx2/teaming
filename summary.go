package teaming

import "sort"

// TeamSummary aggregates one team's membership.
type TeamSummary struct {
	Team   int // 1-indexed team number
	People int // number of people on the team
	Groups int // number of distinct input groups on the team
}

// Summarize aggregates assignments into one TeamSummary per team,
// sorted ascending by Team number. People counts every assignment for the
// team; Groups counts the distinct Group values on the team.
func Summarize(assignments []Assignment) []TeamSummary {
	if len(assignments) == 0 {
		return []TeamSummary{}
	}

	type teamData struct {
		people int
		groups map[string]struct{}
	}

	data := make(map[int]*teamData)
	for _, a := range assignments {
		td, ok := data[a.Team]
		if !ok {
			td = &teamData{groups: make(map[string]struct{})}
			data[a.Team] = td
		}
		td.people++
		td.groups[a.Group] = struct{}{}
	}

	summaries := make([]TeamSummary, 0, len(data))
	for team, td := range data {
		summaries = append(summaries, TeamSummary{
			Team:   team,
			People: td.people,
			Groups: len(td.groups),
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Team < summaries[j].Team
	})

	return summaries
}
