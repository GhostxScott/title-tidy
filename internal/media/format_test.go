package media

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// First pass: core table tests for each exported formatter.

func TestFormatShowName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "Empty", input: "", want: ""},
		{name: "NoYearRemovesTags", input: "My.Cool.Show.720p.x264", want: "My Cool Show"},
		{name: "SingleYear", input: "Some.Show.2024.1080p.WEB-DL.x264", want: "Some Show (2024)"},
		{name: "YearRangeTakesFirst", input: "Cool-Show-2023-2024-1080p", want: "Cool Show (2023)"},
		{name: "TagsBeforeYearAreRemoved", input: "Great.Show.1080p.2022.x265", want: "Great Show (2022)"},
		{name: "SpacingCleanup", input: "The----Show....2021", want: "The Show (2021)"},
		{name: "AfterYearDiscarded", input: "Show.Name.2024.Extra.Stuff.1080p", want: "Show Name (2024)"},
		{name: "YearRangeSpaceSeparator", input: "Another.Show 2021 2022 720p", want: "Another Show (2021)"},
		{name: "PlainNoChange", input: "Plain Show", want: "Plain Show"},
		{name: "AlreadyFormattedYear", input: "Some Film (2022)", want: "Some Film (2022)"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := FormatShowName(tc.input)
			if got != tc.want {
				t.Errorf("FormatShowName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatSeasonName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "CanonicalSeason", input: "Season 1", want: "Season 01"},
		{name: "ShortS", input: "s2", want: "Season 02"},
		{name: "SeasonWithPrefix", input: "S03 - Something", want: "Season 03"},
		{name: "SimpleNumber", input: "5", want: "Season 05"},
		{name: "AltSeparator", input: "Season_11 Extras", want: "Season 11"},
		{name: "NotFound", input: "Extras", want: ""},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := FormatSeasonName(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FormatSeasonName(%q) mismatch (-want +got)\n%s", tc.input, diff)
			}
		})
	}
}

// For episode formatting we exercise both video and subtitle paths using the direct SxxExx pattern.
func TestFormatEpisodeName_DirectPattern(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "VideoBasic", input: "Show.Name.S01E02.1080p.mkv", want: "S01E02.mkv"},
		{name: "VideoUpperExt", input: "Show.Name.S01E02.1080p.MKV", want: "S01E02.MKV"},
		{name: "SubtitleLangShort", input: "Show.Name.S01E02.en.srt", want: "S01E02.en.srt"},
		{name: "SubtitleLangRegion", input: "Show.Name.S01E03.en-US.srt", want: "S01E03.en-US.srt"},
		{name: "SubtitleNoLang", input: "Show.Name.S01E04.srt", want: "S01E04.srt"},
		{name: "Subtitle3CharLang", input: "Show.Name.S01E05.eng.srt", want: "S01E05.eng.srt"},
		{name: "LowercasePattern", input: "show.name.s01e06.mkv", want: "S01E06.mkv"},
		{name: "AltPattern1x02", input: "Show.Name.1x07.mkv", want: "S01E07.mkv"},
		{name: "DottedPatternPadded", input: "1.04.1080p.mkv", want: "S01E04.mkv"},
		{name: "DottedPatternUnpaddedEpisode", input: "2.4.720p.mkv", want: "S02E04.mkv"},
		{name: "DottedPatternDoublePadded", input: "01.04.some.tag.mkv", want: "S01E04.mkv"},
		{name: "DottedPatternSeason10", input: "10.12.mkv", want: "S10E12.mkv"},
		{name: "DottedPatternRejectYear", input: "2024.05.Doc.mkv", want: ""}, // 2024 season too large -> rejected
		{name: "NoMatch", input: "RandomFile.mkv", want: ""},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Parent context not needed for direct SxxExx pattern; pass nil.
			got := FormatEpisodeName(tc.input, nil)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FormatEpisodeName(%q) mismatch (-want +got)\n%s", tc.input, diff)
			}
		})
	}
}
