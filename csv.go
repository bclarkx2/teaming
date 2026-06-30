package teaming

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ReadPeople reads a CSV from r where the first row is a header.
// It locates the "person" and "group" columns by name (case-insensitive,
// trimmed). Both columns are required. Any other columns (including an
// existing "team" column) are silently ignored. Fully blank rows are skipped.
func ReadPeople(r io.Reader) ([]Person, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true

	header, err := cr.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("teaming/csv: empty input, no header row")
	}
	if err != nil {
		return nil, fmt.Errorf("teaming/csv: reading header: %w", err)
	}

	personCol := -1
	groupCol := -1

	for i, h := range header {
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "person":
			personCol = i
		case "group":
			groupCol = i
		}
	}

	if personCol < 0 {
		return nil, fmt.Errorf("teaming/csv: required column %q not found in header", "person")
	}
	if groupCol < 0 {
		return nil, fmt.Errorf("teaming/csv: required column %q not found in header", "group")
	}

	var people []Person
	rowNum := 1 // header was row 1; data rows start at 2

	for {
		rowNum++
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("teaming/csv: reading row %d: %w", rowNum, err)
		}

		// Skip fully empty rows.
		allEmpty := true
		for _, field := range row {
			if strings.TrimSpace(field) != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			continue
		}

		if personCol >= len(row) {
			return nil, fmt.Errorf("teaming/csv: row %d: too few columns (need column index %d for \"person\")", rowNum, personCol)
		}
		if groupCol >= len(row) {
			return nil, fmt.Errorf("teaming/csv: row %d: too few columns (need column index %d for \"group\")", rowNum, groupCol)
		}

		people = append(people, Person{
			Name:  strings.TrimSpace(row[personCol]),
			Group: strings.TrimSpace(row[groupCol]),
		})
	}

	return people, nil
}

// WriteAssignments writes assignments to w as CSV with header person,group,team.
// The team field is the 1-indexed integer from Assignment.Team. The writer is
// flushed before returning.
func WriteAssignments(w io.Writer, assignments []Assignment) error {
	cw := csv.NewWriter(w)

	if err := cw.Write([]string{"person", "group", "team"}); err != nil {
		return fmt.Errorf("teaming/csv: writing header: %w", err)
	}

	for _, a := range assignments {
		row := []string{a.Person, a.Group, strconv.Itoa(a.Team)}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("teaming/csv: writing assignment for %q: %w", a.Person, err)
		}
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("teaming/csv: flushing: %w", err)
	}

	return nil
}
