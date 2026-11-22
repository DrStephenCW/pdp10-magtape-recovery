#!/usr/bin/env python3
"""Extract PDP-10 TOPS-10 BACKUP and TOPS-20 DUMPER tape files.

Input files are raw tape-file dumps (one per filemark) in core-dump mode:
each 36-bit word stored in 5 bytes (32 bits + low 4 bits).

Usage: pdp10x.py backup|dumper OUTDIR TAPEFILE...
"""
import datetime, json, os, re, sys

def words(b):
    return [(b[i] << 28) | (b[i+1] << 20) | (b[i+2] << 12) | (b[i+3] << 4) | (b[i+4] & 15)
            for i in range(0, len(b) - 4, 5)]

def coredump(ws):
    out = bytearray()
    for w in ws:
        out += bytes(((w >> 28) & 255, (w >> 20) & 255, (w >> 12) & 255, (w >> 4) & 255, w & 15))
    return bytes(out)

def ascii7(ws, nbytes=None):
    out = bytearray()
    for w in ws:
        for sh in (29, 22, 15, 8, 1):
            out.append((w >> sh) & 127)
    if nbytes is not None and 0 < nbytes <= len(out):
        out = out[:nbytes]
    return bytes(out)

def asciz(ws):
    return ascii7(ws).split(b'\0')

def bytes8(ws, nbytes):
    out = bytearray()
    for w in ws:
        for sh in (28, 20, 12, 4):
            out.append((w >> sh) & 255)
    return bytes(out[:nbytes]) if 0 < nbytes <= len(out) else bytes(out)

def udt(w):
    """TOPS-10/20 universal date-time: LH days since 1858-11-17, RH fraction of day."""
    days, frac = w >> 18, w & 0o777777
    if days == 0 or days > 80000:
        return None
    try:
        return datetime.datetime(1858, 11, 17) + datetime.timedelta(days=days, seconds=frac * 86400 / 262144)
    except OverflowError:
        return None

BINEXT = {'EXE', 'SAV', 'REL', 'BIN', 'SHR', 'LOW', 'HGH', 'DMP', 'SVE', 'UNV', 'OVL', 'SYM'}

def is_text(data):
    data = data.replace(b'\0', b'')
    if len(data) < 2:
        return False
    ok = sum(1 for c in data if c in (8, 9, 10, 11, 12, 13) or 32 <= c < 127)
    return ok / len(data) > 0.98 and data.count(10) >= len(data) / 400

BAD = re.compile(r'[<>:"/\\|?*\x00-\x1f]')

def safe(s):
    s = BAD.sub('_', s).strip().rstrip('. ')
    return s or '_'

def settime(path, dt):
    if dt:
        try:
            t = (dt - datetime.datetime(1970, 1, 1)).total_seconds()
            os.utime(path, (t, t))
        except (OSError, OverflowError, ValueError):
            pass

def write_file(dirpath, name, data, dt, used):
    """Write a file; returns (path, duplicate_of_existing)."""
    os.makedirs(dirpath, exist_ok=True)
    base = name
    n = 1
    while (dirpath, name.lower()) in used:
        if used[(dirpath, name.lower())] == hash(data):
            return os.path.join(dirpath, name), True
        n += 1
        name = '%s.~%d~' % (base, n)
    used[(dirpath, name.lower())] = hash(data)
    p = os.path.join(dirpath, name)
    with open(p, 'wb') as f:
        f.write(data)
    settime(p, dt)
    return p, False

def render(data_words, bsiz, leng, ext=''):
    """Return (bytes, kind)."""
    if bsiz == 7:
        t = ascii7(data_words, leng).replace(b'\0', b'')
        return t, 'text'
    if bsiz == 8:
        return bytes8(data_words, leng), 'bytes8'
    t = ascii7(data_words)
    if ext.upper() not in BINEXT and is_text(t):
        return t.replace(b'\0', b''), 'text(36-bit file, looked like ASCII)'
    return coredump(data_words), 'binary(36-bit core-dump, 5 bytes/word)'

# ---------------------------------------------------------------- BACKUP

BREC = 544

def backup(outdir, path, used, manifest):
    b = open(path, 'rb').read()
    n = len(b) // BREC // 5
    label = ''
    ssdir = None
    cur = None
    errors = []
    def flush():
        nonlocal cur
        if not cur:
            return
        name = cur['name']
        data, kind = render(cur['data'], cur['bsiz'], cur['leng'], name['ext'])
        dpath = os.path.join(ssdir, *[safe(c) for c in cur['dirs']])
        fname = safe(name['name'] + ('.' + name['ext'] if name['ext'] else ''))
        p, dup = write_file(dpath, fname, data, cur['date'], used)
        manifest.append({
            'saveset': label, 'tape_file': os.path.basename(path),
            'original': cur['orig'], 'output': os.path.relpath(p, outdir).replace('\\', '/'),
            'kind': kind, 'byte_size': cur['bsiz'], 'length': cur['leng'],
            'words_on_tape': len(cur['data']), 'written': cur['date'].isoformat(' ') if cur['date'] else None,
            'complete': cur['eof'], 'duplicate': dup,
        })
        cur = None
    for r in range(n):
        ws = words(b[r * BREC * 5:(r + 1) * BREC * 5])
        typ, seq, rtnm, flg, chk, siz, lnd = ws[:7]
        if typ == 2 and ssdir is None:
            i = 32
            sysname = ''
            while i < 32 + lnd:
                bt, bl = ws[i] >> 18, ws[i] & 0o777777
                if bl == 0:
                    break
                txt = b' '.join(x for x in asciz(ws[i+1:i+bl]) if x).decode('ascii', 'replace').strip()
                if bt == 5:
                    label = txt
                elif bt == 4:
                    sysname = txt
                i += bl
            sdate = udt(ws[12])
            idx = int(re.sub(r'\D', '', os.path.basename(path)) or 0)
            ssdir = os.path.join(outdir, safe('%02d %s' % (idx, label)) if label else '%02d' % idx)
            manifest_ss.append({'tape_file': os.path.basename(path), 'label': label, 'system': sysname,
                                'saved': sdate.isoformat(' ') if sdate else None,
                                'folder': os.path.relpath(ssdir, outdir)})
        elif typ == 4:
            if flg & 0o040000000000:  # GF$SOF start of file
                flush()
                dirs, dev, raw, nm = [], '', [], {'name': '', 'ext': '', 'gen': ''}
                attrs = [0] * 32
                i = 32
                while i < 32 + lnd:
                    bt, bl = ws[i] >> 18, ws[i] & 0o777777
                    if bl == 0:
                        break
                    if bt == 1:
                        j = i + 1
                        while j < i + bl:
                            st, sl = ws[j] >> 18, ws[j] & 0o777777
                            if sl == 0:
                                break
                            parts = [x.decode('ascii', 'replace') for x in asciz(ws[j+1:j+sl]) if x]
                            if st == 1:
                                dev = ''.join(parts)
                            elif st == 2:
                                nm['name'] = ''.join(parts)
                            elif st == 3:
                                nm['ext'] = ''.join(parts)
                            elif st == 5:
                                nm['gen'] = ''.join(parts)
                            elif st >= 32:
                                for p_ in parts:
                                    m = re.fullmatch(r'(\d+)_(\d+)', p_)
                                    if m:
                                        dirs.append('[%s,%s]' % m.groups())
                                        raw.append('%s,%s' % m.groups())
                                    else:
                                        dirs += p_.split('.')
                                        raw.append(p_)
                            j += sl
                    elif bt == 2:
                        attrs = ws[i+1:i+bl] + [0] * 32
                    i += bl
                if raw and re.fullmatch(r'\d+,\d+', raw[0]):
                    dspec = '[%s]' % ','.join(raw)
                else:
                    dspec = ''.join('<%s>' % d for d in raw)
                orig = (dev + ':' if dev else '') + dspec \
                    + nm['name'] + '.' + nm['ext'] + ('.' + nm['gen'] if nm['gen'] else '')
                if dev:
                    dirs.insert(0, dev)
                bsiz = attrs[6] if attrs[6] else 36
                cur = {'dirs': dirs, 'name': nm, 'orig': orig, 'bsiz': bsiz, 'leng': attrs[5],
                       'date': udt(attrs[2]), 'data': [], 'eof': False}
            if cur is None:
                errors.append('record %d: file data without start-of-file' % r)
                continue
            off = 32 + lnd
            cur['data'] += ws[off:off + siz]
            if flg & 0o400000000000:  # GF$EOF
                cur['eof'] = True
                flush()
    flush()
    return errors

# ---------------------------------------------------------------- DUMPER

DREC = 518

def dumper(outdir, path, used, manifest):
    b = open(path, 'rb').read()
    n = len(b) // DREC // 5
    cur = None
    errors = []
    def flush():
        nonlocal cur
        if not cur:
            return
        pages = cur['pages']
        maxp = max(pages) if pages else -1
        dw = []
        for p in range(maxp + 1):
            dw += pages.get(p, [0] * 512)
        bsiz = cur['bsiz']
        if bsiz == 36 or bsiz == 0:
            dw = dw[:cur['size']] if cur['size'] else dw
            data, kind = render(dw, 36, 0, cur['spec'].split('.')[1] if '.' in cur['spec'] else '')
        else:
            data, kind = render(dw, bsiz, cur['size'])
        spec = cur['spec']
        m = re.match(r'([^:]*):<([^>]*)>([^;]*)', spec)
        dev, d, fn = m.groups() if m else ('', '', spec)
        parts = fn.split('.')
        fname = safe('.'.join(parts[:2]))
        p, dup = write_file(os.path.join(outdir, safe(dev), *[safe(x) for x in d.split('.')]), fname, data, cur['date'], used)
        manifest.append({'saveset': '', 'tape_file': os.path.basename(path), 'original': spec,
                         'output': os.path.relpath(p, outdir).replace('\\', '/'), 'kind': kind,
                         'byte_size': bsiz, 'length': cur['size'], 'pages_on_tape': len(pages),
                         'written': cur['date'].isoformat(' ') if cur['date'] else None, 'complete': cur['eof'], 'duplicate': dup})
        cur = None
    for r in range(n):
        ws = words(b[r * DREC * 5:(r + 1) * DREC * 5])
        typ = ws[4] - (1 << 36) if ws[4] & (1 << 35) else ws[4]
        d = ws[6:]
        if typ == -2:      # file header
            flush()
            spec = asciz(d[:0o200])[0].decode('ascii', 'replace')
            fdb = d[0o200:]
            cur = {'spec': spec, 'bsiz': (fdb[0o11] >> 24) & 0o77, 'size': fdb[0o12],
                   'date': udt(fdb[0o14]), 'pages': {}, 'eof': False}
        elif typ == 0 and cur:
            cur['pages'][ws[3] & 0o777777] = d[:512]
        elif typ == -3 and cur:  # file trailer
            cur['eof'] = True
            flush()
        elif typ == -1:
            manifest_ss.append({'tape_file': os.path.basename(path),
                                'label': asciz(d[3:0o200])[0].decode('ascii', 'replace') if len(d) > 3 else '',
                                'saved': (udt(d[2]).isoformat(' ') if udt(d[2]) else None)})
    flush()
    return errors

manifest_ss = []

if __name__ == '__main__':
    mode, outdir, files = sys.argv[1], sys.argv[2], sys.argv[3:]
    os.makedirs(outdir, exist_ok=True)
    used, manifest, errs = {}, [], {}
    for f in files:
        if os.path.getsize(f) == 0:
            continue
        e = (backup if mode == 'backup' else dumper)(outdir, f, used, manifest)
        if e:
            errs[os.path.basename(f)] = e
    with open(os.path.join(outdir, 'manifest.json'), 'w') as fh:
        json.dump({'savesets': manifest_ss, 'files': manifest, 'errors': errs}, fh, indent=1)
    kinds = {}
    for m in manifest:
        if m['duplicate']:
            kinds['duplicate'] = kinds.get('duplicate', 0) + 1
            continue
        k = m['kind'].split('(')[0]
        kinds[k] = kinds.get(k, 0) + 1
    print('savesets', len(manifest_ss), 'files', len(manifest), kinds,
          'incomplete', sum(1 for m in manifest if not m['complete']), 'errors', errs)
