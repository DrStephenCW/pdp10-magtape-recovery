// runoff2md translates DEC RUNOFF source (.RNO, .RNH, .RND, .RNM, ...) into
// Markdown. It aims to be faithful to the text and its structure rather than
// to RUNOFF's page layout.
//
// Usage:
//
//	runoff2md FILE...            write FILE.md next to each FILE
//	runoff2md -r DIR             convert every RUNOFF file under DIR
//	runoff2md -o out.md FILE     write a single file elsewhere ("-" for stdout)
//	runoff2md -report r.md ...   also write a per-file conversion report
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var runoffExt = regexp.MustCompile(`(?i)\.RN[A-Z]$`)

type result struct {
	path  string
	stats *Stats
	err   error
}

func main() {
	recurse := flag.Bool("r", false, "treat arguments as directories and convert every RUNOFF file beneath them")
	out := flag.String("o", "", "output file for a single input (\"-\" for stdout)")
	report := flag.String("report", "", "write a Markdown conversion report to this file")
	dry := flag.Bool("n", false, "dry run: convert but do not write .md files")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: runoff2md [-r] [-n] [-o out.md] [-report report.md] FILE|DIR...\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	var inputs []string
	for _, a := range flag.Args() {
		if !*recurse {
			inputs = append(inputs, a)
			continue
		}
		filepath.WalkDir(a, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && runoffExt.MatchString(p) {
				inputs = append(inputs, p)
			}
			return nil
		})
	}
	sort.Strings(inputs)
	if *out != "" && len(inputs) != 1 {
		fmt.Fprintln(os.Stderr, "runoff2md: -o needs exactly one input file")
		os.Exit(2)
	}

	var results []result
	failed := false
	for _, in := range inputs {
		src, err := os.ReadFile(in)
		if err != nil {
			results = append(results, result{in, nil, err})
			failed = true
			continue
		}
		md, st := Convert(string(src))
		dest := in + ".md"
		if *out != "" {
			dest = *out
		}
		if !*dry {
			if dest == "-" {
				os.Stdout.WriteString(md)
			} else if err := os.WriteFile(dest, []byte(md), 0o644); err != nil {
				results = append(results, result{in, st, err})
				failed = true
				continue
			}
		}
		results = append(results, result{in, st, nil})
	}
	if *report != "" {
		if err := os.WriteFile(*report, []byte(buildReport(results)), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "runoff2md:", err)
			failed = true
		}
	}
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "runoff2md: %s: %v\n", r.path, r.err)
		}
	}
	if *out != "-" {
		fmt.Fprintf(os.Stderr, "runoff2md: converted %d file(s)\n", len(results))
	}
	if failed {
		os.Exit(1)
	}
}

func buildReport(rs []result) string {
	handled, ignored, unknown := map[string]int{}, map[string]int{}, map[string]int{}
	for _, r := range rs {
		if r.stats == nil {
			continue
		}
		for k, v := range r.stats.Handled {
			handled[k] += v
		}
		for k, v := range r.stats.Ignored {
			ignored[k] += v
		}
		for k, v := range r.stats.Unknown {
			unknown[k] += v
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# runoff2md conversion report\n\n%d file(s) converted.\n\n", len(rs))
	section := func(title string, m map[string]int) {
		fmt.Fprintf(&b, "## %s\n\n", title)
		if len(m) == 0 {
			b.WriteString("None.\n\n")
			return
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return m[keys[i]] > m[keys[j]] || (m[keys[i]] == m[keys[j]] && keys[i] < keys[j]) })
		b.WriteString("| Directive | Uses |\n|---|---|\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "| .%s | %d |\n", strings.TrimPrefix(k, "."), m[k])
		}
		b.WriteString("\n")
	}
	section("Translated directives", handled)
	section("Ignored layout directives", ignored)
	section("Unknown directives (line kept as text)", unknown)
	b.WriteString("## Per-file warnings\n\n")
	any := false
	for _, r := range rs {
		if r.err != nil {
			fmt.Fprintf(&b, "- `%s`: error: %v\n", r.path, r.err)
			any = true
			continue
		}
		if r.stats == nil || len(r.stats.Warnings) == 0 {
			continue
		}
		any = true
		fmt.Fprintf(&b, "- `%s`\n", filepath.ToSlash(r.path))
		for _, w := range r.stats.Warnings {
			fmt.Fprintf(&b, "  - %s\n", w)
		}
	}
	if !any {
		b.WriteString("None.\n")
	}
	return b.String()
}
