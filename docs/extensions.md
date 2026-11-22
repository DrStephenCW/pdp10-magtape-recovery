# File extensions on the decoded PDP-10 tapes

Covers the files decoded from tapes 1, 2 and 4: 1,012 unique files with 100 distinct extensions. Counts are per tape (t1 / t2 / t4). The meanings combine DEC conventions with what was seen inside sample files. Anything marked **(guess)** is an inference, not a known convention.

Tape 2 is the TOPS-10 system at UC Irvine (1975–80). Tape 4 is TOPS-20 at USC Information Sciences Institute (ISI), from accounts named WILLSON (1981). Several files name **Stephen Willson** as author: grad-school essays, the "social impacts" paper, and mail addressed to "Steve" or "WILLSON,S".

## Standard DEC system extensions

| Ext | t2 | t4 | Meaning |
|---|---|---|---|
| RNO | 125 | 27 | RUNOFF source. RUNOFF was DEC's text formatter (`.p`, `.title` commands). Its output was usually `.MEM`. |
| MEM | 52 | 15 | "Memo". Formatted RUNOFF output, ready to print. |
| RNH | 4 | | RUNOFF source for a help file (output `.HLP`). |
| RND / RNM | 1 / 1 | | RUNOFF source that produces `.DOC` / `.MEM`. Both are from Hedrick's Pascal kit. |
| HLP | 5 | 2 | Help text shown by the `HELP` command or a program's `/HELP` switch. |
| DOC | 5 | 1 | Documentation. |
| TXT | 36 | 10 | Plain text. |
| REL | 27 | 28 | Relocatable object code, the output of MACRO, FORTRAN, BLISS and so on. LINK loads it. |
| EXE | 32 | 21 | Executable image (TOPS-10 6.x and TOPS-20). Tape 1's only file is an `.EXE`. |
| SAV | 23 | | Older TOPS-10 saved core image, from before `.EXE` existed. |
| SHR | 1 | | Sharable high segment of a two-segment TOPS-10 program. |
| BIN | 8 | | Binary output, here from the Pascal translator and microcode tools. |
| UNV | 3 | | MACRO-10 UNIVERSAL file: precompiled symbol definitions loaded with `SEARCH`. |
| MAC | 32 | 1 | MACRO-10 assembler source. |
| LST | 3 | 1 | Compiler or assembler listing. |
| MIC | 52 | | MIC (Macro Interpreted Commands). These are TOPS-10 command scripts run with `.DO`. Example: `RESTOR.MIC` mounts a tape and runs BACKUP. |
| CTL | 1 | | Batch control file, submitted with `SUBMIT` to run as a batch job. |
| LOG | 18 | 1 | Batch job logs and other log output. |
| CCL | 6 | 1 | Indirect command file, read as `@file` by COMPIL, LINK, PIP and others. |
| CMD | 1 | 10 | TOPS-20 indirect command file (`@file` or `TAKE`). |
| INI | 21 | 14 | Initialization file. On TOPS-10 the classic one is `SWITCH.INI`, which held default switches for each program. `MLISP.INI` is MLISP's startup file. |
| USR | 21 | | `ACCESS.USR`, the TOPS-10 file-daemon access list. It let other users read your files (`DSKB:MICAS.SAV=[*,*]/EXEC`). |
| STD | 5 | | `ACCESS.STD` / `SWITCH.STD` (guess: standard copies that students in the ICS 145 accounts were given to install). |
| BAK | 8 | | Backup copy written by an editor (SOS or TECO) on save. |
| OLD | 4 | | Previous version kept by hand (convention). |
| TMP | | 4 | Temporary file. The `Z.TMP` and `JRM.TMP` files hold patch or position notes (guess). |
| SCM | 1 | | SRCCOM output: a source-compare listing between two files. |
| DIR | 2 | | Saved directory listing. |
| DAT | 11 | 2 | Data. |
| FRM | 1 | | A blank form (`CONLOG.FRM` is a console log sheet for operators). |
| COM | 1 | | `WASTE.COM` is an accounting usage report (connect time, CPU, disk). The extension is that program's choice (guess). |
| BWR | 1 | | "Beware" file: DEC's convention for known problems and caveats in a release. |
| OPR | 1 | | Operator or installation instructions. |
| RUN | 1 | 2 | TOPS-20 EXEC command script (`def`, `comp`, `load`) (guess: run with `TAKE`). |
| EXP | 3 | | Binary pieces of the Pascal runtime library (guess). |

## Programming languages

| Ext | t2 | t4 | Language |
|---|---|---|---|
| LSP | 21 | 11 | Lisp source. UCI Lisp was Irvine's Lisp. |
| MLI | 52 | 15 | MLISP: Lisp written in Algol-like syntax (`EXPR MORPH(V); BEGIN …`). |
| LAP | 10 | | Lisp Assembly Program: compiled Lisp as PDP-10 instructions. |
| SAM / AMB / NFX | 1 each | | Versions of an Ada grammar written as Lisp `DEFPROP`s. The headers read "STILL AMBIGUOUS", "AMBIGUOUS" and "STILL NOT COMPLETELY FIXED". |
| PAS | 6 | 12 | Pascal (Hedrick's Pascal for TOPS-10/20). |
| BLI | | 21 | BLISS, DEC's systems language (BLISS-10/36). |
| HDR | | 2 | BLISS header pulled in with `REQUIRE` (terminal-handling macros). |
| SIM | 23 | | SIMULA source (DEC SIMULA-10). |
| ALG | 2 | | ALGOL-10 source. |
| SNO | 9 | | SNOBOL source, run with SITBOL. |
| FOR | | 4 | FORTRAN-10 source (TOPS-20 Fortran). |
| FTN | 7 | | FORTRAN for the PDP-11. These are DEC's 1977 Adventure port for FORTRAN IV-PLUS. |
| ODL | 1 | | RSX-11 Task Builder overlay description, for the PDP-11 Adventure build. |
| DEF | 1 | | Adventure data table (`ADVENT.DEF`). |
| TM | 5 | | TREEMETA metacompiler grammar (`.META UCSD`). |
| LR | | 4 | LR(1) grammars for Ada (1981 INRIA standard, ISI Formal Semantics Group). |

## Document preparation (TOPS-20, Scribe)

| Ext | t4 | Meaning |
|---|---|---|
| MSS | 2 | Scribe manuscript source (`@make(text)`, `@device(penguin)`). |
| AUX / OTL / ERR | 1 each | Scribe auxiliary files: cross-references, outline, error log. |
| PRS | 1 | Scribe output for the output device the manuscript named ("penguin") (guess). |
| DIC | 5 (+1 on t2) | Personal spelling dictionary for the SPELL program. |
| EXC | 1 (t2) | Word list with page references made while proofing `SOCIAL.RNO` (guess). |

## TOPS-20 mail and editor settings (tape 4)

| Ext | Count | Meaning |
|---|---|---|
| INIT | 5 | `MM.INIT` and `EMACS.INIT`: settings for the MM mail reader and TOPS-20 EMACS. |
| DATA | 2 | `MSG.DATA`: last-read dates for bulletin boards (BBOARD and others). |

## Hardware and microprogramming (tape 2, Varian V75 work)

| Ext | t2 | Meaning |
|---|---|---|
| DAS | 16 | Varian V70-series DAS assembler source (course assignments for ICS 155). |
| V75 | 3 | Varian V75 configuration files (`SYSGEN.V75` lists the attached equipment). |
| WCS | 1 | Writable control store microcode for the V75. |
| SRC | 3 | Microcode source with line numbers. |
| KOD | 2 | Assembled microcode as hex words (guess: the microassembler's object output). |
| ADR | 2 | Microassembler address file. |
| CON | 2 | Configuration file for the MICSIM microcode simulator. |
| ATR | 3 | Binary attribute tables for `LEX` (guess). |
| P1 | 1 | Hex dump of the ICSLAB floppy disk, 26 May 1978. |
| 68 | 1 | Motorola 6800 cross-assembler listing of MIKBUG. |
| COD | 1 | Program for a simple teaching machine (`LDI`, `STORE`, `JUMP`). |

## Course work and personal conventions (tape 2 unless noted)

| Ext | Count | Meaning |
|---|---|---|
| 145 | 27 | ICS 145 course files (handouts, tests, notes dated April 1980). |
| 001–010 | 10 | Numbered test programs for ICS 145 Assignment 2. |
| QRZ | 8 | Transcripts of "Pseudo-Steve" sessions, a Lisp program that answered questions about office hours and the course (guess: saved logs). |
| NAM / BIG | 7 / 1 | Grad-school application essays. Each `.NAM` defines one school's name (Berkeley, CMU, MIT…); `CARNEG.BIG` is the full essay "By Stephen Willson". |
| PIC | 4 (t4) | ASCII-art banners and pictures. |
| F / FIX | 2 / 2 | Notes on the UCI "F" editor. `SOCIAL.FIX` is a corrected draft of the paper. |
| ME | 1 | `TYPE.ME`: a prank note telling the owner they had left their account logged in. |
| TO | 2 | `HOW.TO`: build instructions for TREEMETA. |
| (none) | 11 / 7 (t4) | No extension, mostly data and notes. |
| !331 | 2 | Not identified. The name looks garbled, or deliberately strange. |

## Conversions to keep in mind
- Nearly all text-type extensions came off the tape as 7-bit ASCII and are now readable text.
- `.EXE`, `.SAV`, `.SHR`, `.REL`, `.UNV`, `.ATR` and `.BIN` are 36-bit binaries. They're kept in 5-byte-per-word format for emulators.
- A few `.REL`, `.SAV` and `.CMD` files were recorded as 7-bit text, so they are probably text files with misleading names (not checked).
