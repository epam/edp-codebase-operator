package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const (
	valuesHeading    = "## Values"
	tableSeparator   = "|--"
	tableRowPrefix   = "|"
	guardEnd         = "{{- end }}"
	rowKeyFieldCount = 3
)

// readme is a generated chart README split at its values table, so the tables of sibling
// variants can be merged while the prose around them is asserted to be identical.
type readme struct {
	guard  string
	header []string
	rows   []string
	footer []string
}

// parseReadme splits generated documentation into the values table and its surroundings.
func parseReadme(doc, guard string) (readme, error) {
	lines := strings.Split(doc, "\n")

	heading := slices.IndexFunc(lines, func(line string) bool {
		return strings.TrimSpace(line) == valuesHeading
	})
	if heading < 0 {
		return readme{}, fmt.Errorf("section %q not found in generated documentation", valuesHeading)
	}

	offset := slices.IndexFunc(lines[heading:], func(line string) bool {
		return strings.HasPrefix(line, tableSeparator)
	})
	if offset < 0 {
		return readme{}, errors.New("values table separator not found in generated documentation")
	}

	start := heading + offset + 1

	end := start
	for end < len(lines) && strings.HasPrefix(lines[end], tableRowPrefix) {
		end++
	}

	if start == end {
		return readme{}, errors.New("values table has no rows")
	}

	return readme{
		guard:  guard,
		header: lines[:start],
		rows:   lines[start:end],
		footer: lines[end:],
	}, nil
}

// mergeReadmes folds the variants of one platform into a single template.
func mergeReadmes(variants []readme) (string, error) {
	if len(variants) == 0 {
		return "", errors.New("no variants to merge")
	}

	base := variants[0]

	for _, v := range variants[1:] {
		if !slices.Equal(v.header, base.header) || !slices.Equal(v.footer, base.footer) {
			return "", errors.New("variants differ outside the values table, which the generator cannot guard")
		}
	}

	rows, err := mergeRows(variants)
	if err != nil {
		return "", err
	}

	merged := make([]string, 0, len(base.header)+len(rows)+len(base.footer))
	merged = append(merged, base.header...)
	merged = append(merged, rows...)
	merged = append(merged, base.footer...)

	return strings.Join(merged, "\n"), nil
}

// mergedRow is a table row together with the variants that document it.
type mergedRow struct {
	key     string
	line    string
	present []bool
}

// mergeRows builds the union of the variants' tables, keeping the alphanumeric order helm-docs
// sorts by so that every variant's own table stays exactly what helm-docs would generate.
func mergeRows(variants []readme) ([]string, error) {
	unique := make([]*mergedRow, 0, len(variants[0].rows))
	byLine := make(map[string]*mergedRow, len(variants[0].rows))

	for i, v := range variants {
		for _, line := range v.rows {
			row, ok := byLine[line]
			if !ok {
				row = &mergedRow{key: rowKey(line), line: line, present: make([]bool, len(variants))}
				byLine[line] = row
				unique = append(unique, row)
			}

			row.present[i] = true
		}
	}

	// Stable, so rows that share a key keep the order of the variants that declare them.
	slices.SortStableFunc(unique, func(a, b *mergedRow) int { return strings.Compare(a.key, b.key) })

	return guardRows(unique, variants)
}

// guardRows wraps each run of variant-specific rows in the guard that selects it.
func guardRows(rows []*mergedRow, variants []readme) ([]string, error) {
	guarded := make([]string, 0, len(rows))
	open := ""

	for _, row := range rows {
		guard, err := rowGuard(row, variants)
		if err != nil {
			return nil, err
		}

		if guard != open {
			if open != "" {
				guarded = append(guarded, guardEnd)
			}

			if guard != "" {
				guarded = append(guarded, guard)
			}

			open = guard
		}

		guarded = append(guarded, row.line)
	}

	if open != "" {
		guarded = append(guarded, guardEnd)
	}

	return guarded, nil
}

func rowGuard(row *mergedRow, variants []readme) (string, error) {
	owners := make([]int, 0, len(variants))

	for i, present := range row.present {
		if present {
			owners = append(owners, i)
		}
	}

	if len(owners) == len(variants) {
		return "", nil
	}

	if len(owners) == 1 {
		return variants[owners[0]].guard, nil
	}

	return "", fmt.Errorf(
		"row %q is documented by %d of %d variants, which needs a combined guard",
		row.key, len(owners), len(variants),
	)
}

// rowKey returns the value path a table row documents, which is what helm-docs sorts by.
func rowKey(line string) string {
	fields := strings.SplitN(line, "|", rowKeyFieldCount)
	if len(fields) < rowKeyFieldCount {
		return line
	}

	return strings.TrimSpace(fields[1])
}
