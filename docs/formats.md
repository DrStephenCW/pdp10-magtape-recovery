# The tape formats, word by word

This is what `tools/pdp10x.py` assumes about each format. Every claim here was
checked against the real tapes: the record headers of all three PDP-10 tapes
parse with no gaps, no partial records and no unknown record types.

Bit numbers follow DEC: bit 0 is the most significant bit of a 36-bit word and
bit 35 the least. "LH" and "RH" are the left and right 18-bit halves.

## How 36 bits got onto 8-bit tape: core-dump mode

A 9-track tape stores 8-bit frames. TOPS-10 and TOPS-20 wrote these tapes in
**core-dump** (also called "industry compatible, 36-bit") mode: each word takes
five frames.

```
frame 1   bits  0-7
frame 2   bits  8-15
frame 3   bits 16-23
frame 4   bits 24-31
frame 5   bits 32-35 in the low 4 bits (high 4 bits are zero)
```

So the first step for every PDP-10 tape file is to read it 5 bytes at a time
and rebuild one 36-bit integer (`words()` in the extractor). Record sizes are
always a multiple of 5 bytes: 2,720 bytes is 544 words, 2,590 bytes is 518.

## Text: five 7-bit characters per word

Text files are usually byte size 7. Each word holds five ASCII characters,
left-justified, and bit 35 is unused (on SOS files it marks a line-number word;
the extractor drops it).

```
| char 1 | char 2 | char 3 | char 4 | char 5 |x|
  0-6      7-13     14-20    21-27    28-34   35
```

The file's byte count (from its attributes) trims the padding at the end. NUL
characters are padding and are removed. Line ends were already CR LF.

Other byte sizes seen: 8-bit files pack four bytes per word, left-justified,
with 4 bits unused. 36-bit files (`.EXE`, `.REL`, `.SAV` …) are kept as raw
words in core-dump form, so a PDP-10 emulator such as SIMH can load them.

A handful of files had byte size 36 but were plainly text (SOS files with line
numbers, for example). If a 36-bit file's extension is not a known binary one
and its 7-bit decode is more than 98% printable with ordinary line lengths, it
is written as text. The manifest marks these as
`text(36-bit file, looked like ASCII)`.

## Dates: the universal date-time word

Both systems store a date and time in one word: LH is the number of days since
**17 November 1858** (the Modified Julian Date epoch), and RH is the fraction of
a day in units of 1/2^18. The extractor sets each output file's modification
time from this, so a directory listing shows the original 1975-1983 dates.

## TOPS-10 BACKUP (tapes 2 and 4)

Tape records are 544 words: a 32-word header, then a variable "non-data" area
of `lnd` words, then up to `siz` words of file data.

| Header word | Name | Use |
|---|---|---|
| 0 | `G$TYPE` | record type: 1 label, 2 start of save set, 3 end of save set, 4 file data, 5 directory, 6 end of volume, 7 comment, 8 continue |
| 1 | `G$SEQ` | record sequence number (checked for gaps in the survey) |
| 2 | `G$RTNM` | relative tape number |
| 3 | `G$FLAG` | bit 0 `GF$EOF` last record of a file; bit 3 `GF$SOF` first record of a file |
| 4 | `G$CHK` | checksum (tape 4 has none; the extractor does not check it) |
| 5 | `G$SIZ` | number of data words in this record |
| 6 | `G$LND` | length of the non-data area |
| 12 | | for a start-of-save-set record: the date the save began |

The non-data area is a list of blocks. Each block starts with a word whose LH
is the block type and RH its length (including that word). The extractor uses:

- **Start of save set (type 2)**: block 5 is the save-set label (for example
  `End of SQ 80 [Granger]`), block 4 the system name (`U C Irvine SY47(4) 70043`).
- **File (type 4), first record**: block 1 is the file name, itself a list of
  sub-blocks (1 device, 2 name, 3 extension, 5 generation, 32 and up the
  directory path, one sub-block per level). A TOPS-10 directory
  `[4776,65]` is stored as the string `4776_65`; a TOPS-20 directory `<WILLSON>`
  as `WILLSON`. Block 2 is the attributes: word 2 the write date, word 5 the
  length in bytes, word 6 the byte size.

File data is the concatenation of each record's `siz` data words from the
`GF$SOF` record through the `GF$EOF` record.

Tape 4 was written on UCI's TOPS-20 system, but in this same BACKUP format (device
names `EO:` and `TS:`, directory `<WILLSON>`), not in DUMPER format. Its three
save sets overlap, so 102 of its 341 files are exact copies; each was written
once.

Tape 2's save sets 30 and 31 recorded no directory names, so two different
files there can have the same name. The second is written as `NAME.EXT.~2~`.

## TOPS-20 DUMPER (tape 1)

Tape records are 518 words: a 6-word header and one 512-word page.

| Header word | Use |
|---|---|
| 3 | RH: page number within the file |
| 4 | record type, as a signed 36-bit number: 0 data page, -1 save-set header, -2 file header, -3 file trailer, -4 tape trailer, -5 user directory info, -6 continued save set |

- **Save-set header (-1)**: page word 2 is the date, the label is ASCIZ from word 3.
- **File header (-2)**: page words 0-0o177 are the full file spec as ASCIZ
  (`PS:<ICS>WHO.EXE.1`); from word 0o200 is the file's FDB. The extractor reads
  FDB word 0o11 bits 6-11 (byte size), 0o12 (byte count) and 0o14 (last write).
- **Data (0)**: the page is placed at its page number, so holes in a file stay
  as zero pages.
- **File trailer (-3)**: the file is complete and gets written.

## Unix tar (tapes 3, 5, 6)

These are ordinary tar archives from 1980s Unix systems.
Bob Chicko extracted them in 2010 when he read the tapes, so there was nothing
to decode. (One file from tape 6, a VAX `a.out` with symbols, was later used to
rebuild lost C source; that work is in
[ada-interpreter-1984](https://github.com/DrStephenCW/ada-interpreter-1984).)
