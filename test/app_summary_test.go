package test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/output"
	"github.com/farhapartex/osql/internal/query"
)

type sizedCatalog struct {
	apps []engine.App
	err  error
}

func (c *sizedCatalog) Apps(context.Context) ([]engine.App, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.apps, nil
}

type realSizer struct {
	sizes  map[string]int64
	unable map[string]bool
	err    error
}

func (s *realSizer) Sizes(ctx context.Context, list []engine.App) error {
	if s.err != nil {
		return s.err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	for i := range list {
		if s.unable[list[i].Name] {
			continue
		}
		list[i].Size = s.sizes[list[i].Name]
		list[i].SizeKnown = true
	}
	return nil
}

func summaryApps() []engine.App {
	return []engine.App{
		{Name: "Chrome", Source: engine.SourceSystem, Modified: stamp(10)},
		{Name: "Safari", Source: engine.SourceMacOS, Modified: stamp(2)},
		{Name: "OrbStack", Source: engine.SourceHomebrew, Modified: stamp(20)},
		{Name: "ripgrep", Source: engine.SourceHomebrewCLI, Modified: stamp(5)},
		{Name: "jq", Source: engine.SourceHomebrewCLI, Modified: stamp(6)},
	}
}

func summarySizes() map[string]int64 {
	return map[string]int64{
		"Chrome":   2_000_000_000,
		"Safari":   500_000_000,
		"OrbStack": 300_000_000,
		"ripgrep":  20_000_000,
		"jq":       5_000_000,
	}
}

func summarizeApps(t *testing.T, catalog engine.Catalog, sizer engine.AppSizer, input string) (engine.AppSummary, error) {
	t.Helper()

	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	executor := engine.NewAppsExecutor(catalog, compiler, sizer)

	stmt, err := query.NewParser(compiler).Parse(mustLex(t, input))
	if err != nil {
		return engine.AppSummary{}, err
	}
	return executor.SummarizeApps(context.Background(), stmt)
}

func TestSummaryAppsSplitsAppsFromTools(t *testing.T) {
	got, err := summarizeApps(t, &sizedCatalog{apps: summaryApps()}, &realSizer{sizes: summarySizes()}, "summary apps")
	if err != nil {
		t.Fatalf("SummarizeApps error = %v", err)
	}

	if got.Apps != 3 {
		t.Errorf("Apps = %d, want 3", got.Apps)
	}
	if got.Tools != 2 {
		t.Errorf("Tools = %d, want 2", got.Tools)
	}
	if got.Total() != 5 {
		t.Errorf("Total() = %d, want 5", got.Total())
	}

	wantApps := int64(2_800_000_000)
	if got.AppsSize != wantApps {
		t.Errorf("AppsSize = %d, want %d", got.AppsSize, wantApps)
	}
	if got.ToolsSize != 25_000_000 {
		t.Errorf("ToolsSize = %d, want 25000000", got.ToolsSize)
	}
	if got.TotalSize() != wantApps+25_000_000 {
		t.Errorf("TotalSize() = %d", got.TotalSize())
	}
}

func TestSummaryAppsIncludesToolsUnlikeAListing(t *testing.T) {
	got, err := summarizeApps(t, &sizedCatalog{apps: summaryApps()}, &realSizer{sizes: summarySizes()}, "summary apps")
	if err != nil {
		t.Fatalf("SummarizeApps error = %v", err)
	}

	found := false
	for _, tally := range got.Sources {
		if tally.Source == engine.SourceHomebrewCLI {
			found = true
			if tally.Count != 2 {
				t.Errorf("homebrew-cli count = %d, want 2", tally.Count)
			}
		}
	}
	if !found {
		t.Error("a summary must account for command-line tools, even though a listing hides them")
	}
}

func TestSummaryAppsSourcesOrderedBySizeDescending(t *testing.T) {
	got, err := summarizeApps(t, &sizedCatalog{apps: summaryApps()}, &realSizer{sizes: summarySizes()}, "summary apps")
	if err != nil {
		t.Fatalf("SummarizeApps error = %v", err)
	}

	want := []string{engine.SourceSystem, engine.SourceMacOS, engine.SourceHomebrew, engine.SourceHomebrewCLI}
	got_ := make([]string, 0, len(got.Sources))
	for _, tally := range got.Sources {
		got_ = append(got_, tally.Source)
	}
	if strings.Join(got_, ",") != strings.Join(want, ",") {
		t.Errorf("sources = %v, want %v", got_, want)
	}
}

func TestSummaryAppsLargestIsTopFiveBySize(t *testing.T) {
	many := []engine.App{}
	sizes := map[string]int64{}
	for i, name := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		many = append(many, engine.App{Name: name, Source: engine.SourceSystem, Modified: stamp(1)})
		sizes[name] = int64(i+1) * 1000
	}

	got, err := summarizeApps(t, &sizedCatalog{apps: many}, &realSizer{sizes: sizes}, "summary apps")
	if err != nil {
		t.Fatalf("SummarizeApps error = %v", err)
	}

	if len(got.Largest) != 5 {
		t.Fatalf("Largest has %d entries, want 5", len(got.Largest))
	}
	want := []string{"g", "f", "e", "d", "c"}
	for i, name := range want {
		if got.Largest[i].Name != name {
			t.Errorf("Largest[%d] = %s, want %s", i, got.Largest[i].Name, name)
		}
	}
}

func TestSummaryAppsModifiedRange(t *testing.T) {
	got, err := summarizeApps(t, &sizedCatalog{apps: summaryApps()}, &realSizer{sizes: summarySizes()}, "summary apps")
	if err != nil {
		t.Fatalf("SummarizeApps error = %v", err)
	}

	if !got.Oldest.Equal(stamp(2)) {
		t.Errorf("Oldest = %v, want %v", got.Oldest, stamp(2))
	}
	if !got.Newest.Equal(stamp(20)) {
		t.Errorf("Newest = %v, want %v", got.Newest, stamp(20))
	}
}

func TestSummaryAppsIgnoresZeroModifiedTimes(t *testing.T) {
	list := []engine.App{
		{Name: "Dated", Source: engine.SourceSystem, Modified: stamp(7)},
		{Name: "Undated", Source: engine.SourceSystem},
	}

	got, err := summarizeApps(t, &sizedCatalog{apps: list}, &realSizer{sizes: map[string]int64{"Dated": 1, "Undated": 2}}, "summary apps")
	if err != nil {
		t.Fatalf("SummarizeApps error = %v", err)
	}

	if !got.Oldest.Equal(stamp(7)) || !got.Newest.Equal(stamp(7)) {
		t.Errorf("range = %v to %v; an app with no date must not widen it", got.Oldest, got.Newest)
	}
}

func TestSummaryAppsCountsUnmeasuredAndExcludesThemFromTotals(t *testing.T) {
	sizer := &realSizer{sizes: summarySizes(), unable: map[string]bool{"Chrome": true}}

	got, err := summarizeApps(t, &sizedCatalog{apps: summaryApps()}, sizer, "summary apps")
	if err != nil {
		t.Fatalf("SummarizeApps error = %v", err)
	}

	if got.Unmeasured != 1 {
		t.Errorf("Unmeasured = %d, want 1", got.Unmeasured)
	}
	if got.AppsSize != 800_000_000 {
		t.Errorf("AppsSize = %d, want 800000000 with Chrome excluded", got.AppsSize)
	}
	for _, app := range got.Largest {
		if app.Name == "Chrome" {
			t.Error("an unmeasured app must not appear under LARGEST")
		}
	}
}

func TestSummaryAppsEmptyCatalog(t *testing.T) {
	got, err := summarizeApps(t, &sizedCatalog{}, &realSizer{}, "summary apps")
	if err != nil {
		t.Fatalf("SummarizeApps error = %v", err)
	}
	if !got.IsEmpty() {
		t.Errorf("IsEmpty() = false for an empty catalog: %+v", got)
	}
}

func TestSummaryAppsPropagatesErrors(t *testing.T) {
	if _, err := summarizeApps(t, &sizedCatalog{err: errors.New("nope")}, &realSizer{}, "summary apps"); err == nil {
		t.Error("a catalog failure must reach the caller")
	}
	if _, err := summarizeApps(t, &sizedCatalog{apps: summaryApps()}, &realSizer{err: errors.New("nope")}, "summary apps"); err == nil {
		t.Error("a sizer failure must reach the caller")
	}
}

func TestSummaryAppsParsing(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	stmt, err := parser.Parse(mustLex(t, "summary apps"))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if stmt.Verb != query.VerbSummary {
		t.Errorf("Verb = %q, want summary", stmt.Verb)
	}
	if stmt.Target != query.TargetApps {
		t.Errorf("Target = %v, want apps", stmt.Target)
	}
	if stmt.Path != "" {
		t.Errorf("Path = %q, want empty", stmt.Path)
	}
}

func TestSummaryAppsRejectsFolderClauses(t *testing.T) {
	tests := []struct {
		input string
		kind  oerr.Kind
	}{
		{"summary apps from 'Documents'", oerr.KindAppsNeedNoPath},
		{"summary apps recursive", oerr.KindAppsNotRecursive},
		{"summary apps where source = 'macos'", oerr.KindSummaryTakesNoWhere},
		{"summary apps nonsense", oerr.KindUnexpectedInput},
	}

	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if _, err := parser.Parse(mustLex(t, tt.input)); !oerr.Is(err, tt.kind) {
				t.Errorf("%s gave %v, want kind %v", tt.input, err, tt.kind)
			}
		})
	}
}

func TestSummaryFromStillWorks(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	stmt, err := query.NewParser(compiler).Parse(mustLex(t, "summary from 'Documents' recursive with skipped"))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if stmt.Target == query.TargetApps {
		t.Error("a folder summary must not be treated as an apps summary")
	}
	if !stmt.Recursive || !stmt.IncludeSkipped {
		t.Errorf("folder summary lost its clauses: %+v", stmt)
	}
}

func TestSourceAliases(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"hmb", engine.SourceHomebrew},
		{"HMB", engine.SourceHomebrew},
		{" hmb ", engine.SourceHomebrew},
		{"brew", engine.SourceHomebrew},
		{"hmb-cli", engine.SourceHomebrewCLI},
		{"brew-cli", engine.SourceHomebrewCLI},
		{"homebrew", engine.SourceHomebrew},
		{"homebrew-cli", engine.SourceHomebrewCLI},
		{"macos", engine.SourceMacOS},
		{"nonsense", "nonsense"},
	}

	for _, tt := range tests {
		if got := engine.CanonicalSource(tt.in); got != tt.want {
			t.Errorf("CanonicalSource(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSourceAliasFiltersLikeTheFullName(t *testing.T) {
	catalog := &fakeCatalog{apps: sampleApps()}

	for _, input := range []string{
		"apps where source = 'hmb'",
		"apps where source = 'brew'",
		"apps where source = 'homebrew'",
	} {
		got := appNames(listApps(t, catalog, input))
		if len(got) != 1 || got[0] != "OrbStack" {
			t.Errorf("%s = %v, want [OrbStack]", input, got)
		}
	}

	for _, input := range []string{
		"apps where source = 'hmb-cli'",
		"apps where source = 'homebrew-cli'",
	} {
		got := appNames(listApps(t, catalog, input))
		if strings.Join(got, ",") != "jq,ripgrep" {
			t.Errorf("%s = %v, want [jq ripgrep]", input, got)
		}
	}
}

func TestSourceColumnShowsCanonicalNameNotAlias(t *testing.T) {
	report := listApps(t, &fakeCatalog{apps: sampleApps()}, "apps where source = 'hmb'")

	var buf bytes.Buffer
	if err := output.NewApps().Render(&buf, report); err != nil {
		t.Fatalf("Render error = %v", err)
	}
	if !strings.Contains(buf.String(), engine.SourceHomebrew) {
		t.Errorf("SOURCE column must show %q, not the alias:\n%s", engine.SourceHomebrew, buf.String())
	}
	if strings.Contains(buf.String(), " hmb ") {
		t.Errorf("alias leaked into the output:\n%s", buf.String())
	}
}

func TestAppSummaryRendererBlocks(t *testing.T) {
	summary := engine.AppSummary{
		Apps:      3,
		AppsSize:  2_800_000_000,
		Tools:     2,
		ToolsSize: 25_000_000,
		Sources: []engine.SourceTally{
			{Source: engine.SourceSystem, Count: 1, Size: 2_000_000_000},
			{Source: engine.SourceHomebrewCLI, Count: 2, Size: 25_000_000},
		},
		Largest: []engine.App{
			{Name: "Chrome", Size: 2_000_000_000},
			{Name: "Safari", Size: 500_000_000},
		},
		Oldest: stamp(2),
		Newest: stamp(20),
	}

	var buf bytes.Buffer
	if err := output.NewAppSummary().Render(&buf, summary); err != nil {
		t.Fatalf("Render error = %v", err)
	}
	got := buf.String()

	for _, want := range []string{
		output.AppsHeading, "WHAT", "COUNT", "SIZE", "SOURCE", "LARGEST", "MODIFIED",
		"apps", "tools", "total", "homebrew-cli", "Chrome", "2026-03-02", "2026-03-20",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestAppSummaryRendererHidesToolRowsWhenThereAreNone(t *testing.T) {
	summary := engine.AppSummary{
		Apps:     1,
		AppsSize: 100,
		Sources:  []engine.SourceTally{{Source: engine.SourceMacOS, Count: 1, Size: 100}},
		Oldest:   stamp(1),
		Newest:   stamp(1),
	}

	var buf bytes.Buffer
	if err := output.NewAppSummary().Render(&buf, summary); err != nil {
		t.Fatalf("Render error = %v", err)
	}
	if strings.Contains(buf.String(), "tools") {
		t.Errorf("a tools row appeared with no tools:\n%s", buf.String())
	}
}

func TestAppSummaryRendererEmptyAndUnmeasured(t *testing.T) {
	var empty bytes.Buffer
	if err := output.NewAppSummary().Render(&empty, engine.AppSummary{}); err != nil {
		t.Fatalf("Render error = %v", err)
	}
	if !strings.Contains(empty.String(), "didn't find any installed apps") {
		t.Errorf("empty summary message missing:\n%s", empty.String())
	}

	var noted bytes.Buffer
	summary := engine.AppSummary{
		Apps:       2,
		AppsSize:   100,
		Sources:    []engine.SourceTally{{Source: engine.SourceSystem, Count: 2, Size: 100}},
		Unmeasured: 2,
		Oldest:     stamp(1),
		Newest:     stamp(1),
	}
	if err := output.NewAppSummary().Render(&noted, summary); err != nil {
		t.Fatalf("Render error = %v", err)
	}
	if !strings.Contains(noted.String(), "2 apps could not be measured") {
		t.Errorf("unmeasured notice missing:\n%s", noted.String())
	}
}

func TestAppSummaryColumnsAlignWithFolderSummary(t *testing.T) {
	var buf bytes.Buffer
	summary := engine.AppSummary{
		Apps:     1,
		AppsSize: 1024,
		Sources:  []engine.SourceTally{{Source: engine.SourceMacOS, Count: 1, Size: 1024}},
		Largest:  []engine.App{{Name: "Safari", Size: 1024}},
		Oldest:   stamp(1),
		Newest:   stamp(1),
	}
	if err := output.NewAppSummary().Render(&buf, summary); err != nil {
		t.Fatalf("Render error = %v", err)
	}

	for _, line := range strings.Split(buf.String(), "\n") {
		if line == "" || line == output.AppsHeading {
			continue
		}
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("every block line is indented two spaces like the folder summary: %q", line)
		}
	}
}
