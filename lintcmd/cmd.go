// Package lintcmd implements the frontend of an analysis runner.
// It serves as the entry-point for the staticcheck command, and can also be used to implement custom linters that behave like staticcheck.
package lintcmd

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sort"
	"strings"
	"sync"
	"time"

	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/config"
	"honnef.co/go/tools/go/loader"
	"honnef.co/go/tools/lintcmd/version"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/buildutil"
)

// Command represents a linter command line tool.
type Command struct {
	name           string
	analyzers      map[string]*lint.Analyzer
	version        string
	machineVersion string

	flags struct {
		fs *flag.FlagSet

		tags         string
		tests        bool
		printVersion bool
		showIgnored  bool
		formatter    string
		explain      string
		listChecks   bool

		debugCpuprofile       string
		debugMemprofile       string
		debugVersion          bool
		debugNoCompileErrors  bool
		debugMeasureAnalyzers string
		debugTrace            string

		checks    list
		fail      list
		goVersion versionFlag
	}
}

// NewCommand returns a new Command.
func NewCommand(name string) *Command {
	cmd := &Command{
		name:           name,
		analyzers:      map[string]*lint.Analyzer{},
		version:        "devel",
		machineVersion: "devel",
	}
	cmd.initFlagSet(name)
	return cmd
}

// SetVersion sets the command's version.
// It is divided into a human part and a machine part.
// For example, Staticcheck 2020.2.1 had the human version "2020.2.1" and the machine version "v0.1.1".
// If you only use Semver, you can set both parts to the same value.
//
// Calling this method is optional. Both versions default to "devel", and we'll attempt to deduce more version information from the Go module.
func (cmd *Command) SetVersion(human, machine string) {
	cmd.version = human
	cmd.machineVersion = machine
}

// FlagSet returns the command's flag set.
// This can be used to add additional command line arguments.
func (cmd *Command) FlagSet() *flag.FlagSet {
	return cmd.flags.fs
}

// AddAnalyzers adds analyzers to the command.
// These are lint.Analyzer analyzers, which wrap analysis.Analyzer analyzers, bundling them with structured documentation.
//
// To add analysis.Analyzer analyzers without providing structured documentation, use AddBareAnalyzers.
func (cmd *Command) AddAnalyzers(as ...*lint.Analyzer) {
	for _, a := range as {
		cmd.analyzers[a.Analyzer.Name] = a
	}
}

// AddBareAnalyzers adds bare analyzers to the command.
func (cmd *Command) AddBareAnalyzers(as ...*analysis.Analyzer) {
	for _, a := range as {
		var title, text string
		if idx := strings.Index(a.Doc, "\n\n"); idx > -1 {
			title = a.Doc[:idx]
			text = a.Doc[idx+2:]
		}

		doc := &lint.Documentation{
			Title:    title,
			Text:     text,
			Severity: lint.SeverityWarning,
		}

		cmd.analyzers[a.Name] = &lint.Analyzer{
			Doc:      doc,
			Analyzer: a,
		}
	}
}

func (cmd *Command) initFlagSet(name string) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	cmd.flags.fs = flags
	flags.Usage = usage(name, flags)

	flags.StringVar(&cmd.flags.tags, "tags", "", "List of `build tags`")
	flags.BoolVar(&cmd.flags.tests, "tests", true, "Include tests")
	flags.BoolVar(&cmd.flags.printVersion, "version", false, "Print version and exit")
	flags.BoolVar(&cmd.flags.showIgnored, "show-ignored", false, "Don't filter ignored problems")
	flags.StringVar(&cmd.flags.formatter, "f", "text", "Output `format` (valid choices are 'stylish', 'text' and 'json')")
	flags.StringVar(&cmd.flags.explain, "explain", "", "Print description of `check`")
	flags.BoolVar(&cmd.flags.listChecks, "list-checks", false, "List all available checks")

	flags.StringVar(&cmd.flags.debugCpuprofile, "debug.cpuprofile", "", "Write CPU profile to `file`")
	flags.StringVar(&cmd.flags.debugMemprofile, "debug.memprofile", "", "Write memory profile to `file`")
	flags.BoolVar(&cmd.flags.debugVersion, "debug.version", false, "Print detailed version information about this program")
	flags.BoolVar(&cmd.flags.debugNoCompileErrors, "debug.no-compile-errors", false, "Don't print compile errors")
	flags.StringVar(&cmd.flags.debugMeasureAnalyzers, "debug.measure-analyzers", "", "Write analysis measurements to `file`. `file` will be opened for appending if it already exists.")
	flags.StringVar(&cmd.flags.debugTrace, "debug.trace", "", "Write trace to `file`")

	cmd.flags.checks = list{"inherit"}
	cmd.flags.fail = list{"all"}
	cmd.flags.goVersion = versionFlag("module")
	flags.Var(&cmd.flags.checks, "checks", "Comma-separated list of `checks` to enable.")
	flags.Var(&cmd.flags.fail, "fail", "Comma-separated list of `checks` that can cause a non-zero exit status.")
	flags.Var(&cmd.flags.goVersion, "go", "Target Go `version` in the format '1.x', or the literal 'module' to use the module's Go version")
}

type list []string

func (list *list) String() string {
	return `"` + strings.Join(*list, ",") + `"`
}

func (list *list) Set(s string) error {
	if s == "" {
		*list = nil
		return nil
	}

	*list = strings.Split(s, ",")
	return nil
}

type versionFlag string

func (v *versionFlag) String() string {
	return fmt.Sprintf("%q", string(*v))
}

func (v *versionFlag) Set(s string) error {
	if s == "module" {
		*v = "module"
	} else {
		var vf lint.VersionFlag
		if err := vf.Set(s); err != nil {
			return err
		}
		*v = versionFlag(s)
	}
	return nil
}

// ParseFlags parses command line flags.
// It must be called before calling Run.
// After calling ParseFlags, the values of flags can be accessed.
//
// Example:
//
// 	cmd.ParseFlags(os.Args[1:])
func (cmd *Command) ParseFlags(args []string) {
	cmd.flags.fs.Parse(args)
}

// Run runs all registered analyzers and reports their findings.
// It always calls os.Exit and does not return.
func (cmd *Command) Run() {
	var measureAnalyzers func(analysis *analysis.Analyzer, pkg *loader.PackageSpec, d time.Duration)
	if path := cmd.flags.debugMeasureAnalyzers; path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatal(err)
		}

		mu := &sync.Mutex{}
		measureAnalyzers = func(analysis *analysis.Analyzer, pkg *loader.PackageSpec, d time.Duration) {
			mu.Lock()
			defer mu.Unlock()
			// FIXME(dh): print pkg.ID
			if _, err := fmt.Fprintf(f, "%s\t%s\t%d\n", analysis.Name, pkg, d.Nanoseconds()); err != nil {
				log.Println("error writing analysis measurements:", err)
			}
		}
	}

	exit := func(code int) {
		if cmd.flags.debugCpuprofile != "" {
			pprof.StopCPUProfile()
		}
		if path := cmd.flags.debugMemprofile; path != "" {
			f, err := os.Create(path)
			if err != nil {
				panic(err)
			}
			runtime.GC()
			pprof.WriteHeapProfile(f)
		}
		if cmd.flags.debugTrace != "" {
			trace.Stop()
		}
		os.Exit(code)
	}
	if path := cmd.flags.debugCpuprofile; path != "" {
		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
	}
	if path := cmd.flags.debugTrace; path != "" {
		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		trace.Start(f)
	}

	if cmd.flags.debugVersion {
		version.Verbose(cmd.version, cmd.machineVersion)
		exit(0)
	}

	cs := make([]*lint.Analyzer, 0, len(cmd.analyzers))
	for _, a := range cmd.analyzers {
		cs = append(cs, a)
	}

	if cmd.flags.listChecks {
		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Analyzer.Name < cs[j].Analyzer.Name
		})
		for _, c := range cs {
			var title string
			if c.Doc != nil {
				title = c.Doc.Title
			}
			fmt.Printf("%s %s\n", c.Analyzer.Name, title)
		}
		exit(0)
	}

	if cmd.flags.printVersion {
		version.Print(cmd.version, cmd.machineVersion)
		exit(0)
	}

	// Validate that the tags argument is well-formed. go/packages
	// doesn't detect malformed build flags and returns unhelpful
	// errors.
	tf := buildutil.TagsFlag{}
	if err := tf.Set(cmd.flags.tags); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("invalid value %q for flag -tags: %s", cmd.flags.tags, err))
		exit(1)
	}

	if explain := cmd.flags.explain; explain != "" {
		check, ok := cmd.analyzers[explain]
		if !ok {
			fmt.Fprintln(os.Stderr, "Couldn't find check", explain)
			exit(1)
		}
		if check.Analyzer.Doc == "" {
			fmt.Fprintln(os.Stderr, explain, "has no documentation")
			exit(1)
		}
		fmt.Println(check.Doc)
		fmt.Println("Online documentation\n    https://staticcheck.io/docs/checks#" + check.Analyzer.Name)
		exit(0)
	}

	var f formatter
	switch cmd.flags.formatter {
	case "text":
		f = textFormatter{W: os.Stdout}
	case "stylish":
		f = &stylishFormatter{W: os.Stdout}
	case "json":
		f = jsonFormatter{W: os.Stdout}
	case "sarif":
		f = &sarifFormatter{}
	case "null":
		f = nullFormatter{}
	default:
		fmt.Fprintf(os.Stderr, "unsupported output format %q\n", cmd.flags.formatter)
		exit(2)
	}

	ps, warnings, err := doLint(cs, cmd.flags.fs.Args(), &options{
		Tags:      cmd.flags.tags,
		LintTests: cmd.flags.tests,
		GoVersion: string(cmd.flags.goVersion),
		Config: config.Config{
			Checks: cmd.flags.checks,
		},
		PrintAnalyzerMeasurement: measureAnalyzers,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		exit(1)
	}

	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}

	var (
		numErrors   int
		numWarnings int
		numIgnored  int
	)

	fail := cmd.flags.fail
	analyzerNames := make([]string, len(cs))
	for i, a := range cs {
		analyzerNames[i] = a.Analyzer.Name
	}
	shouldExit := filterAnalyzerNames(analyzerNames, fail)
	shouldExit["staticcheck"] = true
	shouldExit["compile"] = true

	if f, ok := f.(complexFormatter); ok {
		f.Start(cs)
	}

	for _, p := range ps {
		if p.Category == "compile" && cmd.flags.debugNoCompileErrors {
			continue
		}
		if p.Severity == severityIgnored && !cmd.flags.showIgnored {
			numIgnored++
			continue
		}
		if shouldExit[p.Category] {
			numErrors++
		} else {
			p.Severity = severityWarning
			numWarnings++
		}
		f.Format(p)
	}
	if f, ok := f.(statter); ok {
		f.Stats(len(ps), numErrors, numWarnings, numIgnored)
	}

	if f, ok := f.(complexFormatter); ok {
		f.End()
	}

	if numErrors > 0 {
		exit(1)
	}
	exit(0)
}

func usage(name string, fs *flag.FlagSet) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] [packages]\n", name)

		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		printDefaults(fs)

		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "For help about specifying packages, see 'go help packages'")
	}
}

// isZeroValue determines whether the string represents the zero
// value for a flag.
//
// this function has been copied from the Go standard library's 'flag' package.
func isZeroValue(f *flag.Flag, value string) bool {
	// Build a zero value of the flag's Value type, and see if the
	// result of calling its String method equals the value passed in.
	// This works unless the Value type is itself an interface type.
	typ := reflect.TypeOf(f.Value)
	var z reflect.Value
	if typ.Kind() == reflect.Ptr {
		z = reflect.New(typ.Elem())
	} else {
		z = reflect.Zero(typ)
	}
	return value == z.Interface().(flag.Value).String()
}

// this function has been copied from the Go standard library's 'flag' package and modified to skip debug flags.
func printDefaults(fs *flag.FlagSet) {
	fs.VisitAll(func(f *flag.Flag) {
		// Don't print debug flags
		if strings.HasPrefix(f.Name, "debug.") {
			return
		}

		var b strings.Builder
		fmt.Fprintf(&b, "  -%s", f.Name) // Two spaces before -; see next two comments.
		name, usage := flag.UnquoteUsage(f)
		if len(name) > 0 {
			b.WriteString(" ")
			b.WriteString(name)
		}
		// Boolean flags of one ASCII letter are so common we
		// treat them specially, putting their usage on the same line.
		if b.Len() <= 4 { // space, space, '-', 'x'.
			b.WriteString("\t")
		} else {
			// Four spaces before the tab triggers good alignment
			// for both 4- and 8-space tab stops.
			b.WriteString("\n    \t")
		}
		b.WriteString(strings.ReplaceAll(usage, "\n", "\n    \t"))

		if !isZeroValue(f, f.DefValue) {
			if T := reflect.TypeOf(f.Value); T.Name() == "*stringValue" && T.PkgPath() == "flag" {
				// put quotes on the value
				fmt.Fprintf(&b, " (default %q)", f.DefValue)
			} else {
				fmt.Fprintf(&b, " (default %v)", f.DefValue)
			}
		}
		fmt.Fprint(fs.Output(), b.String(), "\n")
	})
}
