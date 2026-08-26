package test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/farhapartex/osql/internal/apps"
	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/output"
	"github.com/farhapartex/osql/internal/query"
)

type fakeCatalog struct {
	apps  []engine.App
	err   error
	calls int
}

func (c *fakeCatalog) Apps(context.Context) ([]engine.App, error) {
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	return c.apps, nil
}

type fakeSource struct {
	name string
	apps []engine.App
	err  error
}

func (s fakeSource) Name() string { return s.name }

func (s fakeSource) Apps(context.Context) ([]engine.App, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.apps, nil
}

type fakeSizer struct {
	mu     sync.Mutex
	sized  []string
	err    error
	unable map[string]bool
}

func (s *fakeSizer) Sizes(ctx context.Context, list []engine.App) error {
	if s.err != nil {
		return s.err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	for i := range list {
		s.mu.Lock()
		s.sized = append(s.sized, list[i].Name)
		s.mu.Unlock()

		if s.unable[list[i].Name] {
			continue
		}
		list[i].Size = int64(len(list[i].Name)) * 1000
		list[i].SizeKnown = true
	}
	return nil
}

func mustLex(t *testing.T, input string) []query.Token {
	t.Helper()

	tokens, err := query.NewLexer().Lex(input)
	if err != nil {
		t.Fatalf("Lex(%q) error = %v", input, err)
	}
	return tokens
}

func stamp(day int) time.Time {
	return time.Date(2026, 3, day, 9, 0, 0, 0, time.UTC)
}

func sampleApps() []engine.App {
	return []engine.App{
		{Name: "Google Chrome", Version: "140.0.1", Source: engine.SourceSystem, ID: "com.google.Chrome", Modified: stamp(1)},
		{Name: "Safari", Version: "18.2", Source: engine.SourceMacOS, ID: "com.apple.Safari", Modified: stamp(2)},
		{Name: "OrbStack", Version: "1.6.1", Source: engine.SourceHomebrew, ID: "dev.orbstack.OrbStack", Modified: stamp(3)},
		{Name: "ripgrep", Version: "14.1.0", Source: engine.SourceHomebrewCLI, Modified: stamp(4)},
		{Name: "jq", Version: "1.7.1", Source: engine.SourceHomebrewCLI, Modified: stamp(5)},
	}
}

func appsExecutorFor(t *testing.T, catalog engine.Catalog) (*engine.AppsExecutor, query.Parser) {
	t.Helper()
	executor, parser, _ := appsExecutorWithSizer(t, catalog, &fakeSizer{})
	return executor, parser
}

func appsExecutorWithSizer(t *testing.T, catalog engine.Catalog, sizer *fakeSizer) (*engine.AppsExecutor, query.Parser, *fakeSizer) {
	t.Helper()

	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	return engine.NewAppsExecutor(catalog, compiler, sizer), query.NewParser(compiler), sizer
}

func listApps(t *testing.T, catalog engine.Catalog, input string) engine.AppReport {
	t.Helper()

	executor, parser := appsExecutorFor(t, catalog)
	stmt, err := parser.Parse(mustLex(t, input))
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", input, err)
	}

	report, err := executor.ListApps(context.Background(), stmt)
	if err != nil {
		t.Fatalf("ListApps(%q) error = %v", input, err)
	}
	return report
}

func listAppsSized(t *testing.T, catalog engine.Catalog, sizer *fakeSizer, input string) (engine.AppReport, error) {
	t.Helper()

	executor, parser, _ := appsExecutorWithSizer(t, catalog, sizer)
	stmt, err := parser.Parse(mustLex(t, input))
	if err != nil {
		return engine.AppReport{}, err
	}
	return executor.ListApps(context.Background(), stmt)
}

func appNames(report engine.AppReport) []string {
	names := make([]string, 0, len(report.Apps))
	for _, app := range report.Apps {
		names = append(names, app.Name)
	}
	return names
}

func TestAppsHidesCommandLineToolsByDefault(t *testing.T) {
	report := listApps(t, &fakeCatalog{apps: sampleApps()}, "apps")

	want := []string{"Google Chrome", "OrbStack", "Safari"}
	if got := appNames(report); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("apps = %v, want %v", got, want)
	}
	if report.Tools != 2 {
		t.Errorf("Tools = %d, want 2", report.Tools)
	}
}

func TestAppsIncludesToolsWhenSourceAsked(t *testing.T) {
	report := listApps(t, &fakeCatalog{apps: sampleApps()}, "apps where source = 'homebrew-cli'")

	want := []string{"jq", "ripgrep"}
	if got := appNames(report); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("apps = %v, want %v", got, want)
	}
	if report.Tools != 0 {
		t.Errorf("Tools = %d, want 0 when the source was asked for", report.Tools)
	}
}

func TestAppsSortedByNameCaseInsensitively(t *testing.T) {
	catalog := &fakeCatalog{apps: []engine.App{
		{Name: "zoom", Source: engine.SourceSystem},
		{Name: "Alfred", Source: engine.SourceSystem},
		{Name: "brew-thing", Source: engine.SourceSystem},
	}}

	want := []string{"Alfred", "brew-thing", "zoom"}
	if got := appNames(listApps(t, catalog, "apps")); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("apps = %v, want %v", got, want)
	}
}

func TestAppsFilters(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"apps", []string{"Google Chrome", "OrbStack", "Safari"}},
		{"apps where name = 'Safari'", []string{"Safari"}},
		{"apps where name != 'Safari'", []string{"Google Chrome", "OrbStack"}},
		{"apps where name_like = '%Chrome%'", []string{"Google Chrome"}},
		{"apps where name_like = 'Orb%'", []string{"OrbStack"}},
		{"apps where source = 'macos'", []string{"Safari"}},
		{"apps where source = 'homebrew'", []string{"OrbStack"}},
		{"apps where version = '18.2'", []string{"Safari"}},
		{"apps where version_like = '1%'", []string{"Google Chrome", "OrbStack", "Safari"}},
		{"apps where version_like = '1.%'", []string{"OrbStack"}},
		{"apps where version_like = '%.2'", []string{"Safari"}},
		{"apps where id = 'com.apple.Safari'", []string{"Safari"}},
		{"apps where id_like = 'com.%'", []string{"Google Chrome", "Safari"}},
		{"apps where name_like = '%a%' and source = 'macos'", []string{"Safari"}},
		{"apps where name_like = '%nothing%'", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := appNames(listApps(t, &fakeCatalog{apps: sampleApps()}, tt.input))
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("%s = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestAppsSourceValueIsCaseAndSpaceTolerant(t *testing.T) {
	for _, input := range []string{
		"apps where source = 'MACOS'",
		"apps where source = ' macos '",
	} {
		got := appNames(listApps(t, &fakeCatalog{apps: sampleApps()}, input))
		if len(got) != 1 || got[0] != "Safari" {
			t.Errorf("%s = %v, want [Safari]", input, got)
		}
	}
}

func TestCountAppsPushesOneRow(t *testing.T) {
	executor, parser := appsExecutorFor(t, &fakeCatalog{apps: sampleApps()})
	stmt, err := parser.Parse(mustLex(t, "count(apps)"))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	sink := &engine.SliceSink{}
	if err := executor.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatalf("Execute error = %v", err)
	}

	if len(sink.Rows) != 1 {
		t.Fatalf("pushed %d rows, want 1", len(sink.Rows))
	}
	if sink.Rows[0].Name != "apps" || sink.Rows[0].Count != 3 {
		t.Errorf("row = %+v, want apps/3", sink.Rows[0])
	}
}

func TestAppsCatalogErrorReachesCaller(t *testing.T) {
	executor, parser := appsExecutorFor(t, &fakeCatalog{err: errors.New("no permission")})
	stmt, _ := parser.Parse(mustLex(t, "apps"))

	if _, err := executor.ListApps(context.Background(), stmt); err == nil {
		t.Fatal("ListApps returned no error when the catalog failed")
	}
}

func TestAppsReadsCatalogOncePerQuery(t *testing.T) {
	catalog := &fakeCatalog{apps: sampleApps()}
	listApps(t, catalog, "apps where name_like = '%a%'")

	if catalog.calls != 1 {
		t.Errorf("catalog read %d times, want 1", catalog.calls)
	}
}

func TestAppsCancelledContextStops(t *testing.T) {
	executor, parser := appsExecutorFor(t, &fakeCatalog{apps: sampleApps()})
	stmt, _ := parser.Parse(mustLex(t, "apps"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := executor.ListApps(ctx, stmt); err == nil {
		t.Error("ListApps ignored a cancelled context")
	}
}

func TestAppsParseErrors(t *testing.T) {
	tests := []struct {
		input string
		kind  oerr.Kind
	}{
		{"apps from 'Documents'", oerr.KindAppsNeedNoPath},
		{"apps from '/Applications'", oerr.KindAppsNeedNoPath},
		{"apps recursive", oerr.KindAppsNotRecursive},
		{"apps recursive where name = 'x'", oerr.KindAppsNotRecursive},
		{"delete apps from '/'", oerr.KindCannotDeleteApps},
		{"delete apps", oerr.KindCannotDeleteApps},
		{"apps where type = 'txt'", oerr.KindFieldNotForTarget},
		{"apps where count(child) > 1", oerr.KindFieldNotForTarget},
		{"apps nonsense", oerr.KindUnexpectedInput},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
			parser := query.NewParser(compiler)

			stmt, err := parser.Parse(mustLex(t, tt.input))
			if err == nil {
				executor := engine.NewAppsExecutor(&fakeCatalog{}, compiler, &fakeSizer{})
				_, err = executor.ListApps(context.Background(), stmt)
			}
			if err == nil {
				t.Fatalf("%s was accepted", tt.input)
			}
			if !oerr.Is(err, tt.kind) {
				t.Errorf("%s gave %v, want kind %v", tt.input, err, tt.kind)
			}
		})
	}
}

func TestFileFieldsRejectAppOnlyFields(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	for _, input := range []string{
		"files from 'Documents' where version = '1'",
		"files from 'Documents' where source = 'macos'",
		"folders from 'Documents' where id = 'x'",
	} {
		if _, err := parser.Parse(mustLex(t, input)); !oerr.Is(err, oerr.KindFieldNotForTarget) {
			t.Errorf("%s gave %v, want field_not_for_target", input, err)
		}
	}
}

func TestAppsParsesToAppsVerb(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	stmt, err := parser.Parse(mustLex(t, "apps"))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if stmt.Verb != query.VerbApps {
		t.Errorf("Verb = %q, want %q", stmt.Verb, query.VerbApps)
	}
	if stmt.Target != query.TargetApps {
		t.Errorf("Target = %v, want apps", stmt.Target)
	}
	if stmt.Path != "" {
		t.Errorf("Path = %q, want empty", stmt.Path)
	}

	counted, err := parser.Parse(mustLex(t, "count(apps)"))
	if err != nil {
		t.Fatalf("Parse count error = %v", err)
	}
	if counted.Verb != query.VerbCount || counted.Target != query.TargetApps {
		t.Errorf("count(apps) gave verb %q target %v", counted.Verb, counted.Target)
	}
}

func TestCatalogMergesSourcesWithoutDuplicates(t *testing.T) {
	bundles := fakeSource{name: "bundles", apps: []engine.App{
		{Name: "OrbStack", Version: "1.6.1", Source: engine.SourceSystem, ID: "dev.orbstack", Modified: stamp(1)},
		{Name: "Safari", Version: "18.2", Source: engine.SourceMacOS},
	}}
	casks := fakeSource{name: "casks", apps: []engine.App{
		{Name: "orbstack", Version: "1.6.1_17010", Source: engine.SourceHomebrew},
		{Name: "ngrok", Version: "3.36.1", Source: engine.SourceHomebrew},
	}}

	catalog := apps.NewCatalog(bundles, casks)
	found, err := catalog.Apps(context.Background())
	if err != nil {
		t.Fatalf("Apps error = %v", err)
	}

	if len(found) != 3 {
		t.Fatalf("got %d apps, want 3 (orbstack merged): %+v", len(found), found)
	}

	byName := map[string]engine.App{}
	for _, app := range found {
		byName[app.Name] = app
	}

	orb, ok := byName["OrbStack"]
	if !ok {
		t.Fatalf("merged app kept the cask name, not the bundle name: %+v", found)
	}
	if orb.Source != engine.SourceHomebrew {
		t.Errorf("OrbStack source = %q, want homebrew", orb.Source)
	}
	if orb.Version != "1.6.1" {
		t.Errorf("OrbStack version = %q, want the bundle's 1.6.1", orb.Version)
	}
	if orb.ID != "dev.orbstack" {
		t.Errorf("OrbStack id = %q, want the bundle's id", orb.ID)
	}
}

func TestCatalogSkipsFailingSource(t *testing.T) {
	catalog := apps.NewCatalog(
		fakeSource{name: "broken", err: errors.New("no such directory")},
		fakeSource{name: "good", apps: []engine.App{{Name: "Safari", Source: engine.SourceMacOS}}},
	)

	found, err := catalog.Apps(context.Background())
	if err != nil {
		t.Fatalf("Apps error = %v; a missing folder must not fail the query", err)
	}
	if len(found) != 1 || found[0].Name != "Safari" {
		t.Errorf("got %+v, want just Safari", found)
	}
}

func TestCatalogDropsUnnamedApps(t *testing.T) {
	catalog := apps.NewCatalog(fakeSource{apps: []engine.App{
		{Name: "", Source: engine.SourceSystem},
		{Name: "   ", Source: engine.SourceSystem},
		{Name: "Real", Source: engine.SourceSystem},
	}})

	found, _ := catalog.Apps(context.Background())
	if len(found) != 1 || found[0].Name != "Real" {
		t.Errorf("got %+v, want just Real", found)
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Google Chrome", "googlechrome"},
		{"google-chrome", "googlechrome"},
		{"OrbStack", "orbstack"},
		{"WezTerm", "wezterm"},
		{"font-meslo-lg-nerd-font", "fontmeslolgnerdfont"},
		{"Python 3.12", "python312"},
		{"", ""},
		{"---", ""},
	}

	for _, tt := range tests {
		if got := apps.NormalizeName(tt.in); got != tt.want {
			t.Errorf("NormalizeName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestAppsDoesNotMeasureSizeUnlessAsked(t *testing.T) {
	sizer := &fakeSizer{}
	report, err := listAppsSized(t, &fakeCatalog{apps: sampleApps()}, sizer, "apps")
	if err != nil {
		t.Fatalf("ListApps error = %v", err)
	}

	if len(sizer.sized) != 0 {
		t.Errorf("measured %v without \"with size\" — sizing is the expensive path", sizer.sized)
	}
	if report.Sized {
		t.Error("Sized = true without \"with size\"")
	}
	for _, app := range report.Apps {
		if app.SizeKnown {
			t.Errorf("%s reported a size that was never asked for", app.Name)
		}
	}
}

func TestAppsWithSizeMeasuresAndTotals(t *testing.T) {
	sizer := &fakeSizer{}
	report, err := listAppsSized(t, &fakeCatalog{apps: sampleApps()}, sizer, "apps with size")
	if err != nil {
		t.Fatalf("ListApps error = %v", err)
	}

	if !report.Sized {
		t.Fatal("Sized = false after \"with size\"")
	}
	if len(sizer.sized) != 3 {
		t.Errorf("measured %v, want the 3 listed apps", sizer.sized)
	}

	var want int64
	for _, app := range report.Apps {
		if !app.SizeKnown {
			t.Errorf("%s has no size", app.Name)
		}
		want += app.Size
	}
	if report.TotalSize != want {
		t.Errorf("TotalSize = %d, want %d", report.TotalSize, want)
	}
}

func TestAppsWithSizeMeasuresOnlyWhatSurvivedTheFilter(t *testing.T) {
	sizer := &fakeSizer{}
	if _, err := listAppsSized(t, &fakeCatalog{apps: sampleApps()}, sizer, "apps with size where name_like = '%Chrome%'"); err != nil {
		t.Fatalf("ListApps error = %v", err)
	}

	if len(sizer.sized) != 1 || sizer.sized[0] != "Google Chrome" {
		t.Errorf("measured %v, want only Google Chrome — filtering must come before sizing", sizer.sized)
	}
}

func TestAppsWithSizeSkipsHiddenTools(t *testing.T) {
	sizer := &fakeSizer{}
	if _, err := listAppsSized(t, &fakeCatalog{apps: sampleApps()}, sizer, "apps with size"); err != nil {
		t.Fatalf("ListApps error = %v", err)
	}

	for _, name := range sizer.sized {
		if name == "ripgrep" || name == "jq" {
			t.Errorf("measured hidden tool %q", name)
		}
	}
}

func TestAppsTotalIgnoresUnmeasurableApps(t *testing.T) {
	sizer := &fakeSizer{unable: map[string]bool{"Safari": true}}
	report, err := listAppsSized(t, &fakeCatalog{apps: sampleApps()}, sizer, "apps with size")
	if err != nil {
		t.Fatalf("ListApps error = %v", err)
	}

	var want int64
	for _, app := range report.Apps {
		if app.Name == "Safari" {
			if app.SizeKnown {
				t.Error("Safari should be unmeasurable in this test")
			}
			continue
		}
		want += app.Size
	}
	if report.TotalSize != want {
		t.Errorf("TotalSize = %d, want %d — an unreadable app must not count as zero-and-known", report.TotalSize, want)
	}
}

func TestAppsSizerErrorReachesCaller(t *testing.T) {
	sizer := &fakeSizer{err: errors.New("permission denied")}
	if _, err := listAppsSized(t, &fakeCatalog{apps: sampleApps()}, sizer, "apps with size"); err == nil {
		t.Fatal("a failing sizer must not be swallowed")
	}
}

func TestWithSizeParseErrors(t *testing.T) {
	tests := []struct {
		input string
		kind  oerr.Kind
	}{
		{"apps with", oerr.KindWithNeedsSize},
		{"apps with skipped", oerr.KindWithNeedsSize},
		{"apps with sizes", oerr.KindWithNeedsSize},
		{"apps where source = 'macos' with size", oerr.KindWithSizeComesFirst},
		{"count(apps) with size", oerr.KindCountHasNoSize},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
			_, err := query.NewParser(compiler).Parse(mustLex(t, tt.input))
			if err == nil {
				t.Fatalf("%s was accepted", tt.input)
			}
			if !oerr.Is(err, tt.kind) {
				t.Errorf("%s gave %v, want kind %v", tt.input, err, tt.kind)
			}
		})
	}
}

func TestWithSizeParsesBeforeWhere(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	stmt, err := query.NewParser(compiler).Parse(mustLex(t, "apps with size where source = 'macos'"))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	if !stmt.WithSize {
		t.Error("WithSize = false")
	}
	if len(stmt.Predicates) != 1 {
		t.Errorf("got %d predicates, want 1", len(stmt.Predicates))
	}
}

func TestSizerMeasuresRealDirectories(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "Thing.app")
	nested := filepath.Join(bundle, "Contents", "MacOS")

	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "top.txt"), []byte("12345"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "deep.bin"), make([]byte, 100), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	list := []engine.App{
		{Name: "Thing", Path: bundle},
		{Name: "Missing", Path: filepath.Join(root, "Nope.app")},
		{Name: "Unpathed"},
	}

	if err := apps.NewSizer().Sizes(context.Background(), list); err != nil {
		t.Fatalf("Sizes error = %v", err)
	}

	if !list[0].SizeKnown || list[0].Size != 105 {
		t.Errorf("Thing size = %d (known %v), want 105 summed recursively", list[0].Size, list[0].SizeKnown)
	}
	if list[1].SizeKnown {
		t.Error("a missing path must report an unknown size, not zero")
	}
	if list[2].SizeKnown {
		t.Error("an app with no path must report an unknown size")
	}
}

func TestSizerIgnoresSymlinksRatherThanFollowingThem(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "Linked.app")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}

	outside := filepath.Join(root, "huge.bin")
	if err := os.WriteFile(outside, make([]byte, 5000), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "real.txt"), []byte("abc"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(bundle, "link.bin")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	list := []engine.App{{Name: "Linked", Path: bundle}}
	if err := apps.NewSizer().Sizes(context.Background(), list); err != nil {
		t.Fatalf("Sizes error = %v", err)
	}

	if list[0].Size != 3 {
		t.Errorf("size = %d, want 3 — a symlink must not pull in its target's bytes", list[0].Size)
	}
}

func TestSizerHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	list := []engine.App{{Name: "Thing", Path: t.TempDir()}}
	if err := apps.NewSizer().Sizes(ctx, list); err == nil {
		t.Error("Sizes ignored a cancelled context")
	}
}

func TestSizerHandlesEmptyList(t *testing.T) {
	if err := apps.NewSizer().Sizes(context.Background(), nil); err != nil {
		t.Errorf("Sizes(nil) error = %v", err)
	}
}

func TestAppsRendererShowsSizeColumnOnlyWhenSized(t *testing.T) {
	sized := engine.AppReport{
		Apps:      []engine.App{{Name: "Safari", Source: engine.SourceMacOS, Size: 1536, SizeKnown: true}},
		Sized:     true,
		TotalSize: 1536,
	}
	var withSize bytes.Buffer
	if err := output.NewApps().Render(&withSize, sized); err != nil {
		t.Fatalf("Render error = %v", err)
	}
	if !strings.Contains(withSize.String(), output.HeaderSize) {
		t.Errorf("SIZE column missing:\n%s", withSize.String())
	}
	if !strings.Contains(withSize.String(), "on disk") {
		t.Errorf("total missing from footer:\n%s", withSize.String())
	}

	plain := engine.AppReport{Apps: []engine.App{{Name: "Safari", Source: engine.SourceMacOS}}}
	var without bytes.Buffer
	if err := output.NewApps().Render(&without, plain); err != nil {
		t.Fatalf("Render error = %v", err)
	}
	if strings.Contains(without.String(), output.HeaderSize) {
		t.Errorf("SIZE column shown when nothing was measured:\n%s", without.String())
	}
	if strings.Contains(without.String(), "on disk") {
		t.Errorf("footer claims a total that was never measured:\n%s", without.String())
	}
}

func TestAppsRendererShowsDashForUnmeasuredApp(t *testing.T) {
	report := engine.AppReport{
		Apps: []engine.App{
			{Name: "Known", Source: engine.SourceSystem, Size: 2048, SizeKnown: true},
			{Name: "Unknown", Source: engine.SourceSystem},
		},
		Sized:     true,
		TotalSize: 2048,
	}

	var buf bytes.Buffer
	if err := output.NewApps().Render(&buf, report); err != nil {
		t.Fatalf("Render error = %v", err)
	}

	lines := strings.Split(buf.String(), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Unknown") && !strings.Contains(line, output.Absent) {
			t.Errorf("unmeasured app must show %q: %q", output.Absent, line)
		}
	}
}

func TestAppsRendererColumnsAndFooter(t *testing.T) {
	var buf bytes.Buffer
	report := engine.AppReport{
		Apps: []engine.App{
			{Name: "Google Chrome", Version: "140.0.1", Source: engine.SourceSystem, Modified: stamp(1)},
			{Name: "Safari", Version: "", Source: engine.SourceMacOS, Modified: stamp(2)},
		},
		Tools: 129,
	}

	if err := output.NewApps().Render(&buf, report); err != nil {
		t.Fatalf("Render error = %v", err)
	}
	got := buf.String()

	for _, want := range []string{"NAME", "VERSION", "SOURCE", "MODIFIED", "Google Chrome", "140.0.1", "2 apps"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, output.Absent) {
		t.Errorf("a missing version must render %q:\n%s", output.Absent, got)
	}
	if !strings.Contains(got, "129 command-line tools") {
		t.Errorf("hidden tools note missing:\n%s", got)
	}
	if !strings.Contains(got, "source = 'homebrew-cli'") {
		t.Errorf("note must name the filter that shows them:\n%s", got)
	}
}

func TestAppsRendererOmitsNoteWhenNothingHidden(t *testing.T) {
	var buf bytes.Buffer
	report := engine.AppReport{Apps: []engine.App{{Name: "Safari", Source: engine.SourceMacOS}}}

	if err := output.NewApps().Render(&buf, report); err != nil {
		t.Fatalf("Render error = %v", err)
	}
	if strings.Contains(buf.String(), "command-line tool") {
		t.Errorf("note shown with nothing hidden:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "1 app\n") {
		t.Errorf("singular footer expected:\n%s", buf.String())
	}
}

func TestAppsRendererAlignsWideNames(t *testing.T) {
	var buf bytes.Buffer
	report := engine.AppReport{Apps: []engine.App{
		{Name: "日本語アプリ", Version: "1.0", Source: engine.SourceSystem, Modified: stamp(1)},
		{Name: "Safari", Version: "18.2", Source: engine.SourceMacOS, Modified: stamp(2)},
	}}

	if err := output.NewApps().Render(&buf, report); err != nil {
		t.Fatalf("Render error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected a header and two rows:\n%s", buf.String())
	}

	first := strings.Index(lines[1], "1.0")
	second := strings.Index(lines[2], "18.2")
	if first == second {
		return
	}
	if output.DisplayWidth(lines[1][:first]) != output.DisplayWidth(lines[2][:second]) {
		t.Errorf("version column not aligned by display width:\n%s", buf.String())
	}
}
