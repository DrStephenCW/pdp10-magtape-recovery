#!/usr/bin/env python3
"""Self-test for tools/pdp10x.py.

Builds tiny synthetic BACKUP and DUMPER tape files (no real tape data is
needed), runs the extractor on them, and checks names, contents and dates.

    python3 tests/test_pdp10x.py
"""
import datetime, json, os, subprocess, sys, tempfile, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
TOOL = os.path.join(HERE, '..', 'tools', 'pdp10x.py')

M36 = (1 << 36) - 1


def coredump(ws):
    """36-bit words -> 5 bytes each (32 high bits, then the low 4 bits)."""
    out = bytearray()
    for w in ws:
        out += bytes(((w >> 28) & 255, (w >> 20) & 255, (w >> 12) & 255, (w >> 4) & 255, w & 15))
    return bytes(out)


def pack7(s):
    """ASCII text -> 36-bit words, five 7-bit characters per word, NUL padded."""
    b = s.encode('ascii')
    b += b'\0' * (-len(b) % 5)
    ws = []
    for i in range(0, len(b), 5):
        w = 0
        for c in b[i:i + 5]:
            w = (w << 7) | c
        ws.append(w << 1)
    return ws


def asciz(s):
    """ASCIZ string: always at least one NUL at the end."""
    return pack7(s + '\0')


def udt(dt):
    """datetime -> universal date-time word (days since 1858-11-17, 18-bit fraction)."""
    d = dt - datetime.datetime(1858, 11, 17)
    return (d.days << 18) | int(d.seconds * 262144 / 86400)


def block(typ, body):
    return [(typ << 18) | (len(body) + 1)] + body


# ------------------------------------------------------------------ BACKUP

def backup_record(typ, seq, flags, lnd_words, data):
    """One 544-word BACKUP record: 32-word header, non-data area, data."""
    rec = [0] * 32
    rec[0], rec[1], rec[3] = typ, seq, flags
    rec[5], rec[6] = len(data), len(lnd_words)
    rec += lnd_words + data
    assert len(rec) <= 544
    return rec + [0] * (544 - len(rec))


def make_backup(text, when):
    label = block(5, asciz('test save set'))
    start = backup_record(2, 1, 0, label, [])
    start[12] = udt(when)
    name = block(1, block(1, asciz('DSKB')) + block(32, asciz('4776_65'))
                    + block(2, asciz('HELLO')) + block(3, asciz('TXT')))
    attrs = [0] * 8
    attrs[2], attrs[5], attrs[6] = udt(when), len(text), 7
    data = pack7(text)
    sof, eof = 0o040000000000, 0o400000000000
    rec = backup_record(4, 2, sof | eof, name + block(2, attrs), data)
    return coredump(start + rec)


# ------------------------------------------------------------------ DUMPER

def dumper_record(typ, page, d):
    rec = [0] * 6
    rec[3], rec[4] = page, typ & M36
    rec += d
    return rec + [0] * (518 - len(rec))


def make_dumper(text, when):
    ss = [0, 0, udt(when)] + asciz('test dumper set')
    spec = asciz('PS:<ICS>HELLO.TXT.1')
    hdr = spec + [0] * (0o200 - len(spec))
    fdb = [0] * 0o20
    fdb[0o11] = 7 << 24
    fdb[0o12] = len(text)
    fdb[0o14] = udt(when)
    data = pack7(text)
    recs = (dumper_record(-1, 0, ss) + dumper_record(-2, 0, hdr + fdb)
            + dumper_record(0, 0, data + [0] * (512 - len(data)))
            + dumper_record(-3, 0, []))
    return coredump(recs)


class ExtractorTest(unittest.TestCase):
    TEXT = 'Hello from 1980.\r\nSecond line.\r\n'
    WHEN = datetime.datetime(1980, 5, 19, 14, 45)

    def run_tool(self, mode, raw):
        tmp = tempfile.mkdtemp()
        tape = os.path.join(tmp, 'file0000')
        with open(tape, 'wb') as f:
            f.write(raw)
        out = os.path.join(tmp, 'out')
        subprocess.run([sys.executable, TOOL, mode, out, tape], check=True, capture_output=True)
        with open(os.path.join(out, 'manifest.json')) as f:
            return out, json.load(f)

    def check(self, out, man, path, original):
        self.assertEqual(len(man['files']), 1)
        f = man['files'][0]
        self.assertEqual(f['original'], original)
        self.assertEqual(f['output'], path)
        self.assertEqual(f['kind'], 'text')
        self.assertTrue(f['complete'])
        p = os.path.join(out, *path.split('/'))
        with open(p, 'rb') as fh:
            self.assertEqual(fh.read().decode('ascii'), self.TEXT)
        mtime = datetime.datetime(1970, 1, 1) + datetime.timedelta(seconds=os.path.getmtime(p))
        self.assertLess(abs((mtime - self.WHEN).total_seconds()), 2)

    def test_backup(self):
        out, man = self.run_tool('backup', make_backup(self.TEXT, self.WHEN))
        self.assertEqual(man['savesets'][0]['label'], 'test save set')
        self.check(out, man, '00 test save set/DSKB/[4776,65]/HELLO.TXT', 'DSKB:[4776,65]HELLO.TXT')

    def test_dumper(self):
        out, man = self.run_tool('dumper', make_dumper(self.TEXT, self.WHEN))
        self.check(out, man, 'PS/ICS/HELLO.TXT', 'PS:<ICS>HELLO.TXT.1')


if __name__ == '__main__':
    unittest.main()
