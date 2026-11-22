package main

import (
	"strings"
	"testing"
)

func TestConvert(t *testing.T) {
	cases := []struct {
		name, in string
		want     []string // substrings that must appear in the output
	}{
		{"underline", "Say ^&hello\\& world", []string{"Say *hello* world"}},
		{"accept and space flags", "AT_&T##rules", []string{"AT&T rules"}},
		{"chained directives", ".b.c;Heading\nbody text", []string{"# Heading", "body text"}},
		{"literal", ".lit\n(defun f (x) x)\n.el", []string{"```\n(defun f (x) x)\n```"}},
		{"header level", ".hl 1Intro\ntext\n.hl 2The effect\nmore", []string{"## 1 Intro", "### 1.1 The effect"}},
		{"list", ".list\n.le\none\n.le\ntwo\n.end list", []string{"1. one", "2. two"}},
		{"hanging indent", ".lm 10\n.i -5\nTERM###Meaning here\n.lm 0", []string{"- **TERM** — Meaning here"}},
		{"comment", ".figure 10.;.blank won't work\nText", []string{"Text"}},
		{"break", "line one\n.br\nline two", []string{"line one\\\nline two"}},
		{"autoparagraph", ".ap\nfirst para\n   second para", []string{"first para\n\nsecond para"}},
		{"line start escape", "In\n.br\n1980. was a year", []string{"1980\\. was"}},
	}
	for _, tc := range cases {
		got, _ := Convert(tc.in)
		for _, w := range tc.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: output missing %q\n--- got ---\n%s", tc.name, w, got)
			}
		}
	}
}

func TestParseDirectiveLine(t *testing.T) {
	ds, unk, ok := parseDirectiveLine(".test page 4.spacing 1.literal")
	if !ok || unk != "" || len(ds) != 3 || ds[2].name != "LITERAL" {
		t.Fatalf("got %+v %q %v", ds, unk, ok)
	}
	if _, _, ok := parseDirectiveLine("... and so on"); ok {
		t.Fatal("ellipsis parsed as a directive")
	}
}
