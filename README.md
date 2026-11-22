# Reading six old magnetic tapes (1975-1988)

_This project was done with Claude Opus 5.5 Medium Effort guided by Stephen Clarke-Willson. The tapes were read by Bob Chicko of TapeConversions.com._

Six reels of 9-track magnetic tape held Steve Willson's (now Stephen
Clarke-Willson's) work from UC Irvine Information and Computer Science department (ICS): 
undergraduate course work on
the UCI DEC-10, 1981 Ada work on the UCI DEC-20, and his graduate-school Unix
home directory. In January 2010 Bob Chicko of TapeConversions.com
(Arlington, Texas), a paid tape-reading service, read all six on an HP 88780 drive at
6,250 bpi. Three of them were Unix tar tapes, and he extracted them. The other
three were in PDP-10 formats, and they sat as raw dumps until 2026.

This repository is how those three were decoded: a description of the formats,
the extractor that was written for them, and a RUNOFF-to-Markdown translator
for the documents that came out. It holds no tape contents. The things found on
the tapes are published separately (see [What came off the tapes](#what-came-off-the-tapes)).

## The six tapes

| Tape | Format | What was on it | Files |
|---|---|---|---|
| 1 | TOPS-20 **DUMPER**, 2,590-byte records | One program, `PS:<ICS>WHO.EXE.1`, saved 16 Mar 1983 | 1 |
| 2 | TOPS-10 **BACKUP**, 2,720-byte records | 33 save sets from the UCI DEC-10, Apr 1978 to Nov 1981: Pascal and TREEMETA work, Varian V75 microcode, Adventure, AI projects, ICS 145 course files, papers | 771 |
| 3 | Unix tar | The MH 6.6 mail handler, 1987-88 | 798 |
| 4 | **BACKUP**, written on TOPS-20 | Three save sets of `<WILLSON>` on the UCI DEC-20, Aug-Sep 1981: Ada grammars (some obtained from USC-ISI), the ADA81-INRIA syntax, BLISS, Scribe documents | 341 (239 unique) |
| 5 | Unix tar | Ada pretty-printer and parser sources, Nov-Dec 1985 | 48 |
| 6 | Unix tar | A home directory, Dec 1982 to May 1984: Ada, Arcturus, mail, thesis drafts, the F editor | 1,639 |

Bob's read log for tape 1 is in [`docs/tape1-read-log-2010.txt`](docs/tape1-read-log-2010.txt):
eight blocks of 2,590 bytes, then two filemarks.

## Why three tapes were never decoded

Bob's reads left each PDP-10 tape as a set of files, `file0000`, `file0001`,
and so on, one for each filemark. A 2026 survey parsed every record header on
those three tapes and found nothing wrong with the media. Every file is an
exact multiple of the record size, the record sequence numbers have no gaps or
repeats, every record type is valid, and every save set has its start and end
records.

So the problem was tooling, not damage. A tar extractor does nothing with a
stream of 36-bit words packed five bytes at a time. The tapes needed a reader
for DEC's two backup formats.

## How they were decoded

There are three layers, described in detail in [`docs/formats.md`](docs/formats.md).

1. **Bytes to words.** The tapes were written in "core-dump" mode: each 36-bit
   PDP-10 word is five 8-bit frames, 32 bits and then the last 4 bits. The
   extractor turns each tape file back into a list of 36-bit integers.
2. **Words to files.** BACKUP records are 544 words (a 32-word header, a block
   list holding the file name and attributes, then data). DUMPER records are
   518 words (a 6-word header and one 512-word page). The extractor walks the
   records, collects each file's data from its start record to its end record,
   and reads its name, byte size, length and write date.
3. **Words to something a modern machine can use.** Text files (byte size 7,
   five characters per word) become ordinary ASCII with CR LF line ends. 8-bit
   files become plain bytes. Executables and object files (`.EXE`, `.SAV`,
   `.REL` …) stay 36-bit, in the same five-bytes-per-word form, so an emulator
   like SIMH can load them. Each file gets back its original PDP-10 write date,
   converted from DEC's "days since 17 November 1858" format.

Results (all files complete, no errors):

| Tape | Save sets | Files | Text | 36-bit binary | 8-bit | Duplicates written once |
|---|---|---|---|---|---|---|
| 1 | 1 | 1 | 0 | 1 | 0 | 0 |
| 2 | 33 | 771 | 681 | 90 | 0 | 0 |
| 4 | 3 | 341 | 189 | 49 | 1 | 102 |

A second run of the extractor over the raw tapes in November 2026 produced
output identical, byte for byte, to the first.

Some things that made it more than a straight reading of the DEC manuals:

- **Tape 4 is BACKUP, not DUMPER**, even though it came from a TOPS-20 system.
  The names inside are TOPS-20 style (`EO:<WILLSON>AADSYS.BLI.1`), and the
  records carry no checksums.
- **Its three save sets overlap.** 102 of the 341 files are exact copies of
  files in another save set. Each is written once, and the manifest says which
  entries were copies.
- **Text in disguise.** Some files are marked as 36-bit but are plainly text
  (SOS files with line numbers, for example). If the extension is not a known
  binary one and the 7-bit decode is almost all printable, it is written as
  text, and the manifest says so.
- **Missing directory names.** Tape 2's save sets 30 and 31 ("End of Undergrad
  Career") recorded no directories, so two different files can share a name.
  The second is written as `NAME.EXT.~2~`.

## The tools

### `tools/pdp10x.py`: BACKUP and DUMPER extractor

Python 3, standard library only, about 290 lines.

```
python3 tools/pdp10x.py backup OUTDIR tape2/file0000 tape2/file0001 ...
python3 tools/pdp10x.py dumper OUTDIR tape1/file0000
```

It writes `OUTDIR/<NN save-set label>/<device>/<directory>/NAME.EXT`. A TOPS-10
directory appears as `[4776,65]`; a TOPS-20 `<A.B>` becomes `A/B`. Characters
Windows doesn't allow in names are replaced with `_`. It also writes
`OUTDIR/manifest.json`, which lists every save set (label, system, date) and
every file (original PDP-10 name, output path, conversion, byte size, length,
date, whether it was complete, whether it was a duplicate).

What it does not do: verify BACKUP checksums, follow a file across two reels,
or handle BACKUP's encrypted or interchange modes. None of these tapes needed
them.

`tests/test_pdp10x.py` builds a tiny BACKUP tape and a tiny DUMPER tape from
scratch, runs the extractor on them and checks the names, text and dates. It
needs no real tape data:

```
python3 tests/test_pdp10x.py
```

### `tools/runoff2md/`: RUNOFF to Markdown

Most of the documents on tape 2 and many on tape 4 were written in DEC
**RUNOFF**, a formatter whose source is text with dot commands (`.P`, `.HL 1`,
`.LIT`). The printed output (`.MEM`) survives for some of them but not all.
`runoff2md` turns the source into Markdown that keeps the structure
(headings, lists, literal blocks, footnotes, underlining) and drops the page
layout. It ran on all 143 RUNOFF files from the tapes with no unknown
directives. Its [README](tools/runoff2md/README.md) lists every directive and
what it becomes.

Go, standard library only:

```
cd tools/runoff2md
go build              # or GOOS=windows go build
go test
./runoff2md -r DIR    # writes NAME.RNO.md beside every .RN? file under DIR
```

`PS.RNO` becomes `PS.RNO.md`, so `PS.RNO` and `PS.RNH` in one folder don't
collide. Source files are never changed.

## Docs

| File | What it is |
|---|---|
| [`docs/formats.md`](docs/formats.md) | Core-dump packing, 7-bit text, the date word, and the BACKUP and DUMPER record layouts as the extractor reads them |
| [`docs/extensions.md`](docs/extensions.md) | The 100 file extensions found on tapes 1, 2 and 4, and what each meant on TOPS-10 and TOPS-20 at UCI |
| [`docs/tape1-read-log-2010.txt`](docs/tape1-read-log-2010.txt) | Bob Chicko's read log for tape 1 |

## What came off the tapes

- [ada-parser-1981](https://github.com/DrStephenCW/ada-parser-1981): an Ada
  parser in MLISP and UCI Lisp, January 1981 (tape 2).
- [ada-interpreter-1984](https://github.com/DrStephenCW/ada-interpreter-1984):
  the Arcturus Ada interpreter, January 1984, rebuilt and running again (tape 6).
- [f-editor-1981](https://github.com/DrStephenCW/f-editor-1981): the `nf`
  screen editor in C, and Frank Tadman's Ada version (tape 6).

## Credits

The tapes were read in 2010 by Bob Chicko of TapeConversions.com, Arlington,
Texas, a service that read old 9-track computer tapes for a fee. Formats
surveyed, extractor and translator written, and this README drafted in 2026 by
Claude Opus 5.5 Medium Effort, guided by Stephen Clarke-Willson. The format
descriptions follow DEC's TOPS-10 BACKUP and TOPS-20 DUMPER documentation,
checked against the tapes themselves.
