# runoff2md

Translates DEC RUNOFF source files (`.RNO`, `.RNH`, `.RND`, `.RNM`, and any
other `.RN?`) into Markdown. It keeps the text and its structure (paragraphs,
headings, lists, literal blocks, emphasis) and drops page layout (margins,
spacing, page numbers, running heads).

Written in Go with no dependencies outside the standard library.

## Usage

```
runoff2md FILE...                 writes FILE.md next to each FILE
runoff2md -r DIR                  converts every .RN? file under DIR
runoff2md -o out.md FILE          writes one file elsewhere ("-" = stdout)
runoff2md -report report.md ...   also writes a conversion report
runoff2md -n ...                  dry run (no .md files written)
```

The output is named after the whole source name (`PS.RNO` becomes
`PS.RNO.md`) so that `PS.RNO` and `PS.RNH` in the same folder don't collide.
Source files are only read, never changed.

Build with `go build` (or `GOOS=windows go build` for a Windows `.exe`).

## What gets translated

| RUNOFF | Markdown |
|---|---|
| Filled text, `.P`, `.B`/`.BLANK`, `.S`/`.SKIP`, blank lines, `.AP` (indented line starts a paragraph) | Paragraphs, re-wrapped at 78 columns |
| `.BR`/`.BREAK` | Hard line break |
| `.HL n text` | Numbered heading (`.HL 1` is `##`, `.HL 2` is `###`, ...) |
| `.CH`, `.AX` | `#` Chapter / Appendix heading |
| `.TITLE` | `#` document title (once) |
| `.C`/`.CENTER` | Centred HTML paragraph. The first centred line of a document with no `.TITLE` becomes its `#` title; a lone centred line between body paragraphs becomes a `##` section heading |
| `.LIT` ... `.EL` | Fenced code block, copied exactly. Literal blocks that are plainly prose (an address, a letter's closing) become lines with hard breaks; a lone right-pushed date becomes a right-aligned paragraph |
| `.NF` ... `.F` | Code block when the lines look like code or aligned columns; otherwise lines with hard breaks. Lines all tabbed to the middle of the page are treated as centred |
| `.LIST` / `.LE` / `.END LIST` | Numbered list (bulleted if `.LIST` names a bullet character), nested |
| `.LM n` with `.I -n` (hanging indent) | Bulleted item; a term set off by `###` or a tab becomes `**TERM** — text` |
| Other indented text (`.LM n`) | Block quote |
| `.NOTE` / `.END NOTE` | Block quote headed **NOTE** |
| `.FN` ... `!` | Markdown footnote |
| `.PAGE` | Horizontal rule |
| `.FIG n` | HTML comment noting the reserved space |
| Flags: `^&...\&` and `&x` (underline), `_x` (accept), `#` (space), `^x`/`\x` (case) | Underlining becomes `*emphasis*`; the rest are applied |
| A single underlined numbered line (`^&1.1 How to use\&`) | Heading at the matching depth |
| `.;` and `.!` comments, chained directives (`.b.c;text`, `.s 1.lm 16.i -16`) | Understood |

Ignored because they only affect the printed page: `.SP`/`.SPACING`,
`.RM`, `.TEST PAGE`, `.PS`, `.NUMBER`, `.HEADER`, `.JUSTIFY`, `.FT`
(first title), `.SUBTITLE` (running head), `.INDEX`/`.X`, `.TAB STOPS`,
`.FLAG ...`. Unknown directives are kept in the output as text and listed in
the report.
