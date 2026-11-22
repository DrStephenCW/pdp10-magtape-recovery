package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Stats records which directives a conversion saw and what it did with them.
type Stats struct {
	Handled  map[string]int // directives translated into Markdown structure
	Ignored  map[string]int // page-layout directives with no Markdown meaning
	Unknown  map[string]int // directives not recognised; their line is kept as text
	Warnings []string
}

func newStats() *Stats {
	return &Stats{Handled: map[string]int{}, Ignored: map[string]int{}, Unknown: map[string]int{}}
}

func (s *Stats) warn(line int, format string, args ...any) {
	s.Warnings = append(s.Warnings, fmt.Sprintf("line %d: ", line)+fmt.Sprintf(format, args...))
}

// Directive table. The key is the canonical name; each alias (abbreviation)
// maps to it. Multi-word commands are written with a single space.
var aliases = map[string]string{
	"BLANK": "BLANK", "B": "BLANK",
	"SKIP": "SKIP", "S": "SKIP",
	"BREAK": "BREAK", "BR": "BREAK",
	"PARAGRAPH": "PARAGRAPH", "PARA": "PARAGRAPH", "P": "PARAGRAPH",
	"INDENT": "INDENT", "I": "INDENT",
	"LEFT MARGIN": "LEFT MARGIN", "LM": "LEFT MARGIN", "LEFT": "LEFT MARGIN",
	"RIGHT MARGIN": "RIGHT MARGIN", "RM": "RIGHT MARGIN", "RIGHT": "RIGHT MARGIN",
	"FILL": "FILL", "F": "FILL",
	"NO FILL": "NO FILL", "NOFILL": "NO FILL", "NF": "NO FILL",
	"JUSTIFY": "JUSTIFY", "J": "JUSTIFY",
	"NO JUSTIFY": "NO JUSTIFY", "NOJUSTIFY": "NO JUSTIFY", "NJ": "NO JUSTIFY",
	"LITERAL": "LITERAL", "LIT": "LITERAL",
	"END LITERAL": "END LITERAL", "EL": "END LITERAL",
	"CENTER": "CENTER", "CENTRE": "CENTER", "C": "CENTER", "CE": "CENTER",
	"AUTOPARAGRAPH": "AUTOPARAGRAPH", "AP": "AUTOPARAGRAPH",
	"NO AUTOPARAGRAPH": "NO AUTOPARAGRAPH", "NO AP": "NO AUTOPARAGRAPH", "NAP": "NO AUTOPARAGRAPH",
	"PAGE": "PAGE", "PG": "PAGE",
	"TEST PAGE": "TEST PAGE", "TP": "TEST PAGE",
	"SPACING": "SPACING", "SP": "SPACING",
	"PAPER SIZE": "PAPER SIZE", "PAGE SIZE": "PAPER SIZE", "PS": "PAPER SIZE",
	"NUMBER": "NUMBER", "NUMBER PAGE": "NUMBER", "NM": "NUMBER",
	"NO NUMBER": "NO NUMBER", "NONUMBER": "NO NUMBER", "NNM": "NO NUMBER",
	"HEADER": "HEADER", "HD": "HEADER",
	"NO HEADER": "NO HEADER", "NOHEADER": "NO HEADER", "NHD": "NO HEADER",
	"FIRST TITLE": "FIRST TITLE", "FT": "FIRST TITLE",
	"TITLE": "TITLE", "T": "TITLE",
	"SUBTITLE": "SUBTITLE", "ST": "SUBTITLE", "SUBTTL": "SUBTITLE",
	"HEADER LEVEL": "HEADER LEVEL", "HL": "HEADER LEVEL",
	"CHAPTER": "CHAPTER", "CH": "CHAPTER",
	"APPENDIX": "APPENDIX", "AX": "APPENDIX",
	"LIST": "LIST", "LS": "LIST",
	"LIST ELEMENT": "LIST ELEMENT", "LE": "LIST ELEMENT",
	"END LIST": "END LIST", "ELS": "END LIST",
	"NOTE": "NOTE", "NT": "NOTE",
	"END NOTE": "END NOTE", "EN": "END NOTE",
	"FIGURE": "FIGURE", "FIG": "FIGURE", "FG": "FIGURE",
	"FOOTNOTE": "FOOTNOTE", "FN": "FOOTNOTE",
	"END FOOTNOTE": "END FOOTNOTE", "EFN": "END FOOTNOTE",
	"INDEX": "INDEX", "X": "INDEX", "Y": "INDEX",
	"DO INDEX": "DO INDEX", "PRINT INDEX": "DO INDEX",
	"TAB STOPS": "TAB STOPS", "TS": "TAB STOPS",
	"UPPER CASE": "UPPER CASE", "UC": "UPPER CASE",
	"LOWER CASE": "LOWER CASE", "LC": "LOWER CASE",
	"PERIOD": "PERIOD", "PR": "PERIOD",
	"NO PERIOD": "NO PERIOD", "NPR": "NO PERIOD",
	"STANDARD": "STANDARD", "SD": "STANDARD",
	"FLAGS": "FLAGS", "FLAGS ALL": "FLAGS", "FL": "FLAGS",
	"NO FLAGS": "NO FLAGS", "NO FLAGS ALL": "NO FLAGS", "NFL": "NO FLAGS",
	"KEEP": "KEEP", "NO KEEP": "KEEP",
	"FLAG": "FLAG SETTING", "NO FLAG": "FLAG SETTING",
	"SET PARAGRAPH": "PARAGRAPH",
	"DISPLAY":       "IGNORED-DISPLAY",
}

// First words that may start a two- or three-word command.
var prefixWords = map[string]bool{
	"NO": true, "END": true, "LEFT": true, "RIGHT": true, "TEST": true,
	"PAPER": true, "PAGE": true, "NUMBER": true, "FIRST": true, "HEADER": true,
	"LIST": true, "DO": true, "PRINT": true, "TAB": true, "UPPER": true,
	"LOWER": true, "FLAGS": true, "SET": true,
}

// Commands whose remainder of the line is text rather than further commands.
var takesText = map[string]bool{
	"TITLE": true, "SUBTITLE": true, "HEADER LEVEL": true, "CHAPTER": true,
	"APPENDIX": true, "NOTE": true, "INDEX": true,
}

var ignoredCmds = map[string]bool{
	"RIGHT MARGIN": true, "JUSTIFY": true, "NO JUSTIFY": true, "TEST PAGE": true,
	"SPACING": true, "PAPER SIZE": true, "NUMBER": true, "NO NUMBER": true,
	"HEADER": true, "NO HEADER": true, "FIRST TITLE": true, "INDEX": true,
	"DO INDEX": true, "TAB STOPS": true, "UPPER CASE": true, "LOWER CASE": true,
	"PERIOD": true, "NO PERIOD": true, "STANDARD": true, "KEEP": true,
	"IGNORED-DISPLAY": true, "FLAG SETTING": true,
}

type directive struct {
	name string   // canonical name
	raw  string   // as written, upper-cased
	args []string // numeric or quoted arguments
	text string   // trailing text (after ';' or for text-taking commands)
	hasT bool     // text was given
}

// parseDirectiveLine splits a line beginning with '.' into its chained
// directives. ok is false when the line is not a directive at all (for
// example "..." or ". text"). unknown is set when a word was not recognised.
func parseDirectiveLine(line string) (ds []directive, unknown string, ok bool) {
	i := 0
	n := len(line)
	for i < n && line[i] == '.' {
		i++ // the dot
		if i >= n {
			return ds, "", len(ds) > 0
		}
		if line[i] == ';' || line[i] == '!' {
			// comment to end of line
			ds = append(ds, directive{name: "COMMENT"})
			return ds, "", true
		}
		if !isLetter(line[i]) {
			return ds, "", len(ds) > 0
		}
		// first word
		j := i
		for j < n && isLetter(line[j]) {
			j++
		}
		w1 := strings.ToUpper(line[i:j])
		name, raw := "", w1
		end := j
		// try multi-word forms
		if prefixWords[w1] {
			k := j
			for k < n && line[k] == ' ' {
				k++
			}
			m := k
			for m < n && isLetter(line[m]) {
				m++
			}
			if m > k {
				w2 := w1 + " " + strings.ToUpper(line[k:m])
				// three-word forms such as NO FLAGS ALL
				k2 := m
				for k2 < n && line[k2] == ' ' {
					k2++
				}
				m2 := k2
				for m2 < n && isLetter(line[m2]) {
					m2++
				}
				if m2 > k2 {
					w3 := w2 + " " + strings.ToUpper(line[k2:m2])
					if c, ok := aliases[w3]; ok {
						name, raw, end = c, w3, m2
					}
				}
				if name == "" {
					if c, ok := aliases[w2]; ok {
						name, raw, end = c, w2, m
					}
				}
			}
		}
		if name == "" {
			c, ok := aliases[w1]
			if !ok {
				if len(ds) == 0 {
					return nil, w1, true
				}
				return ds, w1, true
			}
			name = c
		}
		d := directive{name: name, raw: raw}
		i = end
		if takesText[name] {
			rest := line[i:]
			if name == "HEADER LEVEL" {
				// level number, then the header text (possibly with no space)
				r := strings.TrimLeft(rest, " \t")
				k := 0
				for k < len(r) && (r[k] == '+' || r[k] == '-' || isDigit(r[k])) {
					k++
				}
				if k > 0 {
					d.args = []string{r[:k]}
				}
				rest = r[k:]
			}
			rest = strings.TrimLeft(rest, " \t")
			rest = strings.TrimPrefix(rest, ";")
			d.text = strings.TrimSpace(rest)
			d.hasT = d.text != ""
			ds = append(ds, d)
			return ds, "", true
		}
		// numeric / quoted arguments
		for {
			for i < n && (line[i] == ' ' || line[i] == '\t' || line[i] == ',') {
				i++
			}
			if i < n && (line[i] == '+' || line[i] == '-' || isDigit(line[i])) {
				k := i + 1
				for k < n && isDigit(line[k]) {
					k++
				}
				d.args = append(d.args, line[i:k])
				i = k
				continue
			}
			if i+2 < n && (line[i] == '\'' || line[i] == '"') && line[i+2] == line[i] {
				d.args = append(d.args, line[i:i+3])
				i += 3
				continue
			}
			break
		}
		if i < n && line[i] == ';' {
			rest := line[i+1:]
			if strings.HasPrefix(rest, ".") && len(rest) > 1 && (isLetter(rest[1]) || rest[1] == ';' || rest[1] == '!') {
				ds = append(ds, d)
				i++
				continue
			}
			d.text = rest
			d.hasT = true
			ds = append(ds, d)
			return ds, "", true
		}
		ds = append(ds, d)
		if i < n && line[i] == '!' {
			return ds, "", true // trailing comment
		}
		if i < n && line[i] != '.' {
			if ignoredCmds[name] {
				return ds, "", true // e.g. ".FLAG MACRO $": arguments we do not model
			}
			// Anything else trailing: RUNOFF would complain. Keep it as text.
			rest := strings.TrimSpace(line[i:])
			if rest != "" {
				ds[len(ds)-1].text = rest
				ds[len(ds)-1].hasT = true
			}
			return ds, "", true
		}
	}
	return ds, "", len(ds) > 0
}

func isLetter(c byte) bool { return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') }
func isDigit(c byte) bool  { return c >= '0' && c <= '9' }

// ---------------------------------------------------------------------------
// Inline flag processing

// segment is a run of text that is either underlined or not.
type segment struct {
	text string
	ul   bool
}

const nbsp = ' ' // RUNOFF's '#' (a space that is never collapsed)

// inline applies RUNOFF's default flags to one input line:
//
//	_x  accept x literally        #   non-collapsing space
//	^x  upper-case x              \x  lower-case x
//	&x  underline x               ^&  underline on   \&  underline off
//
// The underline lock carries across lines, so it lives in *ul.
func inline(s string, ul *bool, flags bool) []segment {
	var segs []segment
	var cur strings.Builder
	curUL := *ul
	emit := func(r rune, u bool) {
		if u != curUL && cur.Len() > 0 {
			segs = append(segs, segment{cur.String(), curUL})
			cur.Reset()
		}
		curUL = u
		cur.WriteRune(r)
	}
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		if !flags {
			emit(c, false)
			continue
		}
		next := rune(0)
		if i+1 < len(rs) {
			next = rs[i+1]
		}
		switch c {
		case '_':
			if next != 0 {
				emit(next, *ul)
				i++
			} else {
				emit('_', *ul)
			}
		case '#':
			emit(nbsp, *ul)
		case '^':
			switch {
			case next == '&':
				*ul = true
				i++
			case unicode.IsLetter(next):
				emit(unicode.ToUpper(next), *ul)
				i++
			default:
				emit('^', *ul)
			}
		case '\\':
			switch {
			case next == '&':
				*ul = false
				i++
			case unicode.IsLetter(next):
				emit(unicode.ToLower(next), *ul)
				i++
			default:
				emit('\\', *ul)
			}
		case '&':
			if next == 0 || next == ' ' || next == '\t' {
				emit('&', *ul)
			} else {
				// underline the single next character (accept-flag aware)
				i++
				if next == '_' && i+1 < len(rs) {
					i++
					next = rs[i]
				}
				emit(next, true)
			}
		default:
			emit(c, *ul)
		}
	}
	if cur.Len() > 0 {
		segs = append(segs, segment{cur.String(), curUL})
	}
	return segs
}

// plainText drops underlining and turns '#' spaces into ordinary spaces.
func plainText(segs []segment) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.text)
	}
	return strings.ReplaceAll(b.String(), string(nbsp), " ")
}

var mdEscaper = strings.NewReplacer(
	`\`, `\\`, "*", `\*`, "_", `\_`, "`", "\\`", "](", `]\(`, "][", `]\[`,
	"<", `\<`, "~", `\~`,
)

var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// renderSegs turns segments into Markdown (or HTML for centred lines).
// Underlined text becomes emphasis, which is what underlining meant on a
// line printer.
func renderSegs(segs []segment, html bool) string {
	var b strings.Builder
	for _, s := range segs {
		esc := mdEscaper.Replace(s.text)
		if html {
			esc = htmlEscaper.Replace(s.text)
		}
		if !s.ul {
			b.WriteString(esc)
			continue
		}
		core := strings.TrimFunc(esc, isSpaceOrNBSP)
		if core == "" {
			b.WriteString(esc)
			continue
		}
		lead := esc[:strings.Index(esc, core)]
		trail := esc[len(lead)+len(core):]
		b.WriteString(lead)
		if html {
			b.WriteString("<em>" + core + "</em>")
		} else {
			b.WriteString("*" + core + "*")
		}
		b.WriteString(trail)
	}
	return b.String()
}

func isSpaceOrNBSP(r rune) bool { return r == ' ' || r == '\t' || r == nbsp }

// ---------------------------------------------------------------------------
// Converter

type listCtx struct {
	ordered bool
	bullet  string
	count   int
}

type converter struct {
	st *Stats

	out       []string // finished Markdown blocks
	footnotes []string

	flags   bool
	fill    bool
	ap      bool
	literal bool
	ul      bool

	lm       int
	lmLevels []int // distinct indented margins seen since lm was last 0
	indent   int   // pending .INDENT / .PARAGRAPH indent for the next paragraph
	hasInd   bool

	// current filled paragraph: logical lines (split by .BREAK) of segments
	para [][]segment

	// no-fill accumulation (raw lines, flags not yet applied)
	nf []string

	// centred lines
	centerNext  int
	centerGroup [][]segment
	centerClose bool // a blank followed the group

	lists    []listCtx
	itemNext bool // the next paragraph starts a list element
	inItem   bool // we are inside a list element (for continuation paragraphs)

	note bool

	inFootnote bool
	fnBuf      [][]segment

	hl          [7]int
	chapter     int
	title       string
	titleDone   bool
	bodyStarted bool
	lastHang    int // level of the last hanging-indent item, 0 if none
	pendingHR   bool
}

func newConverter() *converter {
	return &converter{st: newStats(), flags: true, fill: true}
}

// Convert translates RUNOFF source to Markdown.
func Convert(src string) (string, *Stats) {
	c := newConverter()
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	lines := strings.Split(src, "\n")
	for n, line := range lines {
		c.line(n+1, cleanLine(line))
	}
	c.flushAll()
	return c.document(), c.st
}

func cleanLine(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return r
		}
		if r == '\f' {
			return -1
		}
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, strings.TrimRight(s, " \t"))
}

func (c *converter) line(n int, s string) {
	if c.literal {
		if strings.HasPrefix(s, ".") {
			ds, _, ok := parseDirectiveLine(s)
			if ok && len(ds) > 0 && ds[0].name == "END LITERAL" {
				c.st.Handled["END LITERAL"]++
				c.endLiteral()
				c.apply(n, ds[1:])
				return
			}
		}
		c.nf = append(c.nf, s)
		return
	}
	if c.inFootnote && strings.HasPrefix(s, "!") {
		c.endFootnote()
		return
	}
	if strings.HasPrefix(s, ".") {
		ds, unknown, ok := parseDirectiveLine(s)
		if ok {
			c.apply(n, ds)
			if unknown != "" {
				c.st.Unknown["."+unknown]++
				c.st.warn(n, "unknown directive %q kept as text", s)
				if len(ds) == 0 {
					c.text(n, s)
				}
			}
			return
		}
	}
	c.text(n, s)
}

func (c *converter) text(n int, s string) {
	if c.inFootnote {
		if strings.TrimSpace(s) != "" {
			c.fnBuf = append(c.fnBuf, inline(strings.TrimSpace(s), &c.ul, c.flags))
		}
		return
	}
	if c.centerNext > 0 {
		c.centerNext--
		c.addCentered(strings.TrimSpace(s))
		return
	}
	if !c.fill {
		c.nf = append(c.nf, s)
		return
	}
	if strings.TrimSpace(s) == "" {
		c.flushPara()
		return
	}
	if c.ap && (s[0] == ' ' || s[0] == '\t') {
		c.flushPara()
	}
	segs := inline(strings.TrimLeft(s, " \t"), &c.ul, c.flags)
	c.finishCenter()
	if len(c.para) == 0 {
		c.para = append(c.para, nil)
	}
	last := len(c.para) - 1
	if len(c.para[last]) > 0 {
		c.para[last] = append(c.para[last], segment{" ", false})
	}
	c.para[last] = append(c.para[last], segs...)
}

func argInt(d directive, i, def int) (int, bool) {
	if i >= len(d.args) {
		return def, false
	}
	v, err := strconv.Atoi(strings.TrimPrefix(d.args[i], "+"))
	if err != nil {
		return def, false
	}
	return v, true
}

func (c *converter) apply(n int, ds []directive) {
	for _, d := range ds {
		if d.name == "COMMENT" {
			continue
		}
		if ignoredCmds[d.name] {
			c.st.Ignored[d.name]++
			if d.name == "TEST PAGE" || d.name == "FIRST TITLE" {
				continue
			}
			continue
		}
		c.st.Handled[d.name]++
		switch d.name {
		case "BLANK", "SKIP":
			if c.fill {
				c.flushPara()
				if c.centerNext == 0 && len(c.centerGroup) > 0 {
					c.centerClose = true
				}
			} else {
				k, _ := argInt(d, 0, 1)
				for j := 0; j < k && j < 10; j++ {
					c.nf = append(c.nf, "")
				}
			}
		case "BREAK":
			if c.fill && len(c.para) > 0 && len(c.para[len(c.para)-1]) > 0 {
				c.para = append(c.para, nil)
			}
		case "PARAGRAPH":
			c.flushPara()
			if v, ok := argInt(d, 0, 0); ok {
				c.indent, c.hasInd = v, true
			}
		case "INDENT":
			c.flushPara()
			v, _ := argInt(d, 0, 5)
			c.indent, c.hasInd = v, true
		case "LEFT MARGIN":
			c.flushPara()
			if len(d.args) == 0 {
				c.lm = 0
			} else {
				a := d.args[0]
				v, _ := argInt(d, 0, 0)
				if a[0] == '+' || a[0] == '-' {
					c.lm += v
				} else {
					c.lm = v
				}
				if c.lm < 0 {
					c.lm = 0
				}
			}
			c.noteMargin()
		case "FILL":
			if !c.fill {
				c.flushNoFill()
			}
			c.fill = true
		case "NO FILL":
			c.flushPara()
			c.fill = false
		case "LITERAL":
			c.flushPara()
			c.flushNoFill()
			c.literal = true
		case "END LITERAL":
			// stray .EL outside a literal: harmless
		case "CENTER":
			c.flushPara()
			c.flushNoFill()
			if d.hasT {
				c.addCentered(strings.TrimSpace(d.text))
			} else {
				k, _ := argInt(d, 0, 1)
				if len(d.args) > 0 && k > 1 && k < 20 {
					// .CENTER n historically meant "centre on column n"; one line
					k = 1
				}
				c.centerNext = 1
			}
			continue
		case "AUTOPARAGRAPH":
			c.ap = true
		case "NO AUTOPARAGRAPH":
			c.ap = false
		case "PAGE":
			c.flushAll()
			if len(c.out) > 0 {
				c.pendingHR = true
			}
		case "TITLE":
			if !c.titleDone && d.hasT {
				c.title = collapseSpaces(plainText(inline(d.text, new(bool), c.flags)))
			}
		case "SUBTITLE":
			// Running page header; no visible Markdown equivalent.
			c.st.Ignored["SUBTITLE (as running head)"]++
			c.st.Handled["SUBTITLE"]--
			if c.st.Handled["SUBTITLE"] == 0 {
				delete(c.st.Handled, "SUBTITLE")
			}
		case "HEADER LEVEL":
			c.flushAll()
			lvl, _ := argInt(d, 0, 1)
			if lvl < 1 {
				lvl = 1
			}
			if lvl > 6 {
				lvl = 6
			}
			c.hl[lvl]++
			for k := lvl + 1; k < len(c.hl); k++ {
				c.hl[k] = 0
			}
			var nums []string
			if c.chapter > 0 {
				nums = append(nums, strconv.Itoa(c.chapter))
			}
			for k := 1; k <= lvl; k++ {
				nums = append(nums, strconv.Itoa(c.hl[k]))
			}
			txt := renderSegs(inline(d.text, new(bool), c.flags), false)
			c.heading(lvl+1, strings.TrimSpace(strings.Join(nums, ".")+" "+txt))
		case "CHAPTER", "APPENDIX":
			c.flushAll()
			c.chapter++
			c.hl = [7]int{}
			label := "Chapter " + strconv.Itoa(c.chapter)
			if d.name == "APPENDIX" {
				label = "Appendix " + string(rune('A'+c.chapter-1))
			}
			txt := renderSegs(inline(d.text, new(bool), c.flags), false)
			if txt != "" {
				label += ": " + txt
			}
			c.heading(1, label)
		case "LIST":
			c.flushPara()
			l := listCtx{ordered: true}
			for _, a := range d.args {
				if len(a) == 3 && (a[0] == '\'' || a[0] == '"') {
					l.ordered, l.bullet = false, string(a[1])
				}
			}
			c.lists = append(c.lists, l)
		case "LIST ELEMENT":
			c.flushPara()
			if len(c.lists) == 0 {
				c.st.warn(n, ".LE outside .LIST; treating as a bulleted item")
				c.lists = append(c.lists, listCtx{ordered: false})
			}
			c.itemNext = true
			if d.hasT {
				c.text(n, d.text)
				d.hasT = false
			}
		case "END LIST":
			c.flushPara()
			if len(c.lists) > 0 {
				c.lists = c.lists[:len(c.lists)-1]
			}
			c.inItem = len(c.lists) > 0
			c.itemNext = false
		case "NOTE":
			c.flushAll()
			c.note = true
			title := "NOTE"
			if d.hasT {
				title = plainText(inline(d.text, new(bool), c.flags))
			}
			c.emit("> **" + mdEscaper.Replace(title) + "**")
		case "END NOTE":
			c.flushAll()
			c.note = false
		case "FIGURE":
			c.flushAll()
			k, _ := argInt(d, 0, 1)
			c.emit(fmt.Sprintf("<!-- RUNOFF .FIGURE: %d blank lines reserved here -->", k))
		case "FOOTNOTE":
			c.inFootnote = true
			c.fnBuf = nil
		case "END FOOTNOTE":
			c.endFootnote()
		case "FLAGS":
			c.flags = true
		case "NO FLAGS":
			c.flags = false
		}
		if d.hasT && !takesText[d.name] {
			c.text(n, d.text)
		}
	}
}

func (c *converter) noteMargin() {
	if c.lm == 0 {
		c.lmLevels = nil
		c.lastHang = 0
		return
	}
	for _, v := range c.lmLevels {
		if v == c.lm {
			return
		}
	}
	c.lmLevels = append(c.lmLevels, c.lm)
	sort.Ints(c.lmLevels)
}

func (c *converter) marginLevel() int {
	for i, v := range c.lmLevels {
		if v == c.lm {
			return i + 1
		}
	}
	return 1
}

func (c *converter) endLiteral() {
	c.literal = false
	if allIndented(c.nf, 24) && !anyCodeLike(c.nf) && strings.TrimSpace(strings.Join(c.nf, "")) != "" {
		// A date or signature pushed to the right margin.
		c.finishCenter()
		var parts []string
		for _, l := range c.nf {
			if t := strings.TrimSpace(l); t != "" {
				parts = append(parts, htmlEscaper.Replace(t))
			}
		}
		c.emit(`<p align="right">` + strings.Join(parts, "<br>\n") + `</p>`)
		c.bodyStarted = true
	} else if proseLiteral(c.nf) {
		c.emitLines(c.nf, false)
	} else {
		c.emitCode(c.nf, false)
	}
	c.nf = nil
}

func (c *converter) endFootnote() {
	c.inFootnote = false
	if len(c.fnBuf) == 0 {
		return
	}
	var parts []string
	for _, l := range c.fnBuf {
		parts = append(parts, renderSegs(l, false))
	}
	c.footnotes = append(c.footnotes, strings.Join(parts, " "))
	ref := fmt.Sprintf("[^%d]", len(c.footnotes))
	if len(c.para) == 0 {
		c.para = append(c.para, nil)
	}
	last := len(c.para) - 1
	c.para[last] = append(c.para[last], segment{"\x00" + ref, false})
	c.fnBuf = nil
}

// ---------------------------------------------------------------------------
// Block output

func (c *converter) emit(block string) {
	if c.pendingHR {
		c.out = append(c.out, "---")
		c.pendingHR = false
	}
	c.emitTitle()
	// Keep everything inside a .NOTE in one block quote.
	if c.note && len(c.out) > 0 && strings.HasPrefix(c.out[len(c.out)-1], "> ") && strings.HasPrefix(block, ">") {
		c.out[len(c.out)-1] += "\n>\n" + block
		return
	}
	c.out = append(c.out, block)
}

func (c *converter) emitTitle() {
	if !c.titleDone && c.title != "" {
		c.out = append(c.out, "# "+mdEscaper.Replace(c.title))
		c.titleDone = true
	}
}

func (c *converter) heading(level int, text string) {
	c.lastHang = 0
	c.emit(strings.Repeat("#", level) + " " + text)
}

func (c *converter) flushAll() {
	c.flushPara()
	c.flushNoFill()
	c.finishCenter()
}

// prefixes returns the first-line and continuation prefixes for a block in
// the current context (note, list, indented margin).
func (c *converter) prefixes(hanging bool) (first, rest string) {
	q := ""
	if c.note {
		q = "> "
	}
	if len(c.lists) > 0 {
		depth := len(c.lists)
		pad := strings.Repeat("   ", depth-1)
		if c.itemNext {
			l := &c.lists[depth-1]
			l.count++
			marker := "-  "
			if l.ordered {
				marker = fmt.Sprintf("%d. ", l.count)
			}
			c.itemNext = false
			c.inItem = true
			return q + pad + marker, q + pad + strings.Repeat(" ", len(marker))
		}
		if c.inItem {
			p := q + pad + "   "
			return p, p
		}
		return q + pad, q + pad
	}
	if c.lm > 0 {
		lvl := c.marginLevel()
		if hanging {
			c.lastHang = lvl
			pad := strings.Repeat("  ", lvl-1)
			return q + pad + "- ", q + pad + "  "
		}
		if c.lastHang > 0 {
			pad := strings.Repeat("  ", c.lastHang)
			return q + pad, q + pad
		}
		return q + "> ", q + "> "
	}
	c.lastHang = 0
	return q, q
}

var numberedHead = regexp.MustCompile(`^(\d+(\.\d+)*)\.?\s+\S`)

func (c *converter) flushPara() {
	if len(c.para) == 0 {
		c.hasInd = false
		return
	}
	// drop empty trailing logical lines
	for len(c.para) > 0 && len(c.para[len(c.para)-1]) == 0 {
		c.para = c.para[:len(c.para)-1]
	}
	if len(c.para) == 0 {
		c.hasInd = false
		return
	}
	c.finishCenter()
	hanging := c.hasInd && c.indent < 0
	c.hasInd = false

	// A paragraph that is a single underlined numbered line ("^&1.1 How to
	// use\&") is a section heading written by hand.
	if len(c.para) == 1 && len(c.lists) == 0 && !c.note && c.lm == 0 {
		segs := trimSegs(c.para[0])
		if len(segs) == 1 && segs[0].ul {
			t := strings.TrimSpace(plainText(segs))
			if m := numberedHead.FindStringSubmatch(t); m != nil && len(t) < 80 {
				lvl := strings.Count(m[1], ".") + 2
				if lvl > 6 {
					lvl = 6
				}
				c.para = nil
				c.heading(lvl, mdEscaper.Replace(t))
				c.bodyStarted = true
				return
			}
		}
	}

	first, rest := c.prefixes(hanging)
	var lines []string
	for li, segs := range c.para {
		txt := renderSegs(segs, false)
		if hanging && li == 0 {
			txt = hangSeparator(txt)
		}
		txt = strings.ReplaceAll(txt, string(nbsp), " ")
		txt = strings.ReplaceAll(txt, "\x00", "")
		txt = collapseSpaces(txt)
		wrapped := wrap(txt, 78)
		for wi := range wrapped {
			wrapped[wi] = escapeLineStart(wrapped[wi])
		}
		if li < len(c.para)-1 && len(wrapped) > 0 {
			wrapped[len(wrapped)-1] += `\`
		}
		lines = append(lines, wrapped...)
	}
	c.para = nil
	for i := range lines {
		if i == 0 {
			lines[i] = first + lines[i]
		} else {
			lines[i] = rest + lines[i]
		}
	}
	c.emit(strings.Join(lines, "\n"))
	c.bodyStarted = true
}

// hangSeparator makes "IPCINF########Make a request" (a term set off by a
// run of '#' spaces or a tab in a hanging indent) read as "**IPCINF** — ...".
func hangSeparator(s string) string {
	rs := []rune(s)
	for i := 0; i < len(rs) && i < 40; i++ {
		run := 0
		for i+run < len(rs) && (rs[i+run] == nbsp || rs[i+run] == '\t') {
			run++
		}
		if run >= 3 || (run >= 1 && rs[i] == '\t') {
			term := strings.TrimSpace(string(rs[:i]))
			body := strings.TrimLeft(string(rs[i:]), string([]rune{nbsp, '\t', ' '}))
			if term == "" || body == "" || strings.HasPrefix(term, "*") {
				return s
			}
			return "**" + term + "** — " + body
		}
		if run > 0 {
			i += run - 1
		}
	}
	return s
}

func trimSegs(segs []segment) []segment {
	var out []segment
	for _, s := range segs {
		if strings.TrimFunc(s.text, isSpaceOrNBSP) == "" && !s.ul {
			continue
		}
		out = append(out, s)
	}
	return out
}

func collapseSpaces(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

func wrap(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) > width {
			lines = append(lines, cur)
			cur = w
		} else {
			cur += " " + w
		}
	}
	return append(lines, cur)
}

var orderedStart = regexp.MustCompile(`^(\d+)([.)])(\s|$)`)

// escapeLineStart stops a wrapped line from being read as Markdown block
// syntax (a heading, list item, quote or rule).
func escapeLineStart(l string) string {
	if l == "" {
		return l
	}
	switch l[0] {
	case '#', '+', '-', '=', '>':
		return `\` + l
	}
	if m := orderedStart.FindStringSubmatchIndex(l); m != nil {
		return l[:m[3]] + `\` + l[m[3]:]
	}
	return l
}

// ---- centred text

func (c *converter) addCentered(s string) {
	segs := inline(s, &c.ul, c.flags)
	if c.centerClose {
		c.renderCenter(false)
	}
	c.centerGroup = append(c.centerGroup, segs)
}

// finishCenter is called when ordinary content follows centred lines.
func (c *converter) finishCenter() {
	if len(c.centerGroup) == 0 {
		return
	}
	c.renderCenter(true)
}

func (c *converter) renderCenter(followedByBody bool) {
	var g [][]segment
	for _, l := range c.centerGroup {
		// drop rule lines ("=======") that only underline a centred title
		if t := strings.TrimSpace(plainText(l)); len(t) >= 3 && strings.Trim(t, "=-_*") == "" {
			continue
		}
		g = append(g, l)
	}
	c.centerGroup = nil
	c.centerClose = false
	if len(g) == 0 {
		return
	}
	// The document's first centred line becomes its title when there is no
	// .TITLE; a lone centred line between body paragraphs is a section head.
	if !c.titleDone && c.title == "" && !c.bodyStarted {
		c.title = strings.TrimSpace(plainText(g[0]))
		c.emitTitle()
		g = g[1:]
		if len(g) == 0 {
			return
		}
	}
	if len(g) == 1 && followedByBody && c.bodyStarted && len([]rune(plainText(g[0]))) < 70 {
		c.heading(2, strings.TrimSpace(renderSegs(g[0], false)))
		return
	}
	var parts []string
	for _, l := range g {
		parts = append(parts, strings.TrimSpace(strings.ReplaceAll(renderSegs(l, true), string(nbsp), "&nbsp;")))
	}
	c.emit(`<p align="center">` + strings.Join(parts, "<br>\n") + `</p>`)
}

// ---- no-fill and literal text

var (
	strongCode = regexp.MustCompile(`[\[\]{}=;<>]|\S( {3,}|\t)\S`)
	parenCode  = regexp.MustCompile(`^[\s(].*[()]`)
)

// codeLike reports whether a no-fill or literal line looks like program text
// or a column-aligned table rather than prose.
func codeLike(l string) bool {
	return strongCode.MatchString(l) || parenCode.MatchString(l)
}

// proseLiteral reports whether a .LITERAL block is really prose that the
// author kept from being filled, such as an address or a letter's closing.
func proseLiteral(lines []string) bool {
	nonblank, indented, lower := 0, 0, false
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		nonblank++
		if codeLike(l) {
			return false
		}
		if l[0] == ' ' || l[0] == '\t' {
			indented++
		}
		if strings.IndexFunc(l, unicode.IsLower) >= 0 {
			lower = true
		}
	}
	return nonblank > 0 && lower && indented*2 < nonblank
}

func (c *converter) flushNoFill() {
	if len(c.nf) == 0 {
		return
	}
	lines := c.nf
	c.nf = nil
	nonblank, codey := 0, 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		nonblank++
		if codeLike(l) {
			codey++
		}
	}
	if nonblank == 0 {
		return
	}
	c.finishCenter()
	if codey*3 >= nonblank {
		c.emitCode(lines, true)
		return
	}
	if allIndented(lines, 16) {
		// Lines pushed to the middle of the page with tabs: centred by hand.
		for _, l := range lines {
			if strings.TrimSpace(l) == "" {
				if len(c.centerGroup) > 0 {
					c.centerClose = true
				}
				continue
			}
			c.addCentered(strings.TrimSpace(l))
		}
		return
	}
	c.emitLines(lines, true)
}

func anyCodeLike(lines []string) bool {
	for _, l := range lines {
		if codeLike(l) {
			return true
		}
	}
	return false
}

func allIndented(lines []string, col int) bool {
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		e := expandTabs(l)
		if len(e)-len(strings.TrimLeft(e, " ")) < col {
			return false
		}
	}
	return true
}

// emitLines writes prose laid out line by line (addresses, title pages,
// verse) as paragraphs with hard line breaks.
func (c *converter) emitLines(lines []string, applyFlags bool) {
	c.finishCenter()
	var blocks [][]string
	var cur []string
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			if len(cur) > 0 {
				blocks = append(blocks, cur)
				cur = nil
			}
			continue
		}
		var t string
		if applyFlags {
			t = renderSegs(inline(strings.TrimSpace(l), &c.ul, c.flags), false)
		} else {
			t = mdEscaper.Replace(strings.TrimSpace(l))
		}
		t = strings.ReplaceAll(t, string(nbsp), " ")
		cur = append(cur, escapeLineStart(collapseSpaces(t)))
	}
	if len(cur) > 0 {
		blocks = append(blocks, cur)
	}
	for _, b := range blocks {
		first, rest := c.prefixes(false)
		for i := range b {
			if i < len(b)-1 {
				b[i] += `\`
			}
			if i == 0 {
				b[i] = first + b[i]
			} else {
				b[i] = rest + b[i]
			}
		}
		c.emit(strings.Join(b, "\n"))
	}
	c.bodyStarted = true
}

// emitCode writes a fenced block. applyFlags is true for no-fill text, where
// RUNOFF still interpreted flags; literal text is copied exactly.
func (c *converter) emitCode(lines []string, applyFlags bool) {
	// trim leading/trailing blank lines
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return
	}
	c.finishCenter()
	fence := "```"
	for _, l := range lines {
		if strings.Contains(l, "```") {
			fence = "~~~~"
		}
	}
	first, rest := c.prefixes(false)
	// Keep code blocks inside lists/notes but never turn them into quotes
	// just because of the left margin.
	if len(c.lists) == 0 && !c.note && c.lastHang == 0 {
		first, rest = "", ""
	}
	pad := rest
	var b []string
	b = append(b, first+fence)
	for _, l := range lines {
		t := l
		if applyFlags {
			t = plainText(inline(l, &c.ul, c.flags))
		}
		t = expandTabs(t)
		if t == "" {
			b = append(b, strings.TrimRight(pad, " "))
		} else {
			b = append(b, pad+t)
		}
	}
	b = append(b, pad+fence)
	c.emit(strings.Join(b, "\n"))
	c.bodyStarted = true
}

func expandTabs(s string) string {
	if !strings.Contains(s, "\t") {
		return s
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			n := 8 - col%8
			b.WriteString(strings.Repeat(" ", n))
			col += n
			continue
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
}

func (c *converter) document() string {
	if !c.titleDone && c.title != "" {
		c.out = append([]string{"# " + mdEscaper.Replace(c.title)}, c.out...)
	}
	var b strings.Builder
	for i, blk := range c.out {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(blk)
	}
	if len(c.footnotes) > 0 {
		b.WriteString("\n")
		for i, f := range c.footnotes {
			fmt.Fprintf(&b, "\n[^%d]: %s", i+1, f)
		}
	}
	b.WriteString("\n")
	return b.String()
}
