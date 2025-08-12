package media

import (
	"fmt"
	"strings"

	"github.com/Digital-Shane/treeview"
)

// FormatShowName applies formatting rules to show names.
// It replaces separators with spaces, removes tags, formats years, and cleans up spacing.
func FormatShowName(name string) string {
	if name == "" {
		return name
	}

	formatted := name
	year := ""

	// First, look for a year or year range in the name
	// Match patterns like "2024", "2024-2025", "2024 2025", etc.
	yearMatches := yearRangeRe.FindStringSubmatch(formatted)

	if len(yearMatches) > 1 {
		// Extract just the first year from the match
		year = yearMatches[1]

		// Find the position of the year match
		yearIndex := strings.Index(formatted, yearMatches[0])
		if yearIndex != -1 {
			// Keep only the part before the year (discard everything after)
			formatted = formatted[:yearIndex]
			// If the truncated portion ends with an opening bracket due to an already
			// formatted name like "Title (2022)", trim it, so we don't duplicate it.
			formatted = strings.TrimRight(formatted, " ([{-_")
		}
	}

	// Replace separators with spaces
	formatted = strings.ReplaceAll(formatted, ".", " ")
	formatted = strings.ReplaceAll(formatted, "-", " ")
	formatted = strings.ReplaceAll(formatted, "_", " ")

	// Remove common encoding tags (in case any remain before the year)
	formatted = encodingTagsRe.ReplaceAllString(formatted, "")

	// Clean up extra spaces
	formatted = strings.TrimSpace(strings.Join(strings.Fields(formatted), " "))

	// Add year in parentheses if we found one
	if year != "" {
		formatted = formatted + " (" + year + ")"
	}

	return formatted
}

// FormatSeasonName extracts season number from input and returns formatted season folder name.
// Returns a standardized season folder name (e.g., "Season 01") if season is found, empty string if not.
func FormatSeasonName(input string) string {
	season, found := ExtractSeasonNumber(input)
	if !found {
		return ""
	}
	return fmt.Sprintf("Season %02d", season)
}

// FormatEpisodeName extracts season and episode numbers from input using node context and returns formatted episode name.
// Returns a standardized episode format (e.g., "S01E02.mp4") if both season and episode are found, empty string if not.
// Preserves the file extension and language codes from the original filename.
func FormatEpisodeName(input string, node *treeview.Node[treeview.FileInfo]) string {
	season, episode, found := ParseSeasonEpisode(input, node)
	if !found {
		return ""
	}

	// Preserve the file extension (with language code for subtitles)
	ext := ""
	if IsSubtitle(input) {
		ext = ExtractSubtitleSuffix(input)
	} else {
		ext = ExtractExtension(input)
	}

	return fmt.Sprintf("S%02dE%02d%s", season, episode, ext)
}
