#!/usr/bin/env python3
"""Inject a TeamSpeak 3 identity string into a TS3 client settings.db (in place).

The identity is stored in the ProtobufItems table, in the item whose top-level
field 17 (the identity container) holds a submessage whose field 1 is the
identity string ("<counter>V<base64>"). The table also keeps a Checksum row =
SHA1(concatenation of all numbered item values in numeric key order), which must
be recomputed after editing.

Usage: inject_identity.py <identity-string> <path-to-settings.db>
"""
import sqlite3, hashlib, sys

if len(sys.argv) != 3:
    sys.exit("usage: inject_identity.py <identity> <settings.db>")
NEW_ID = sys.argv[1].encode()
DB = sys.argv[2]
NEW_NICK = b"MrFree"


def read_varint(b, i):
    shift = res = 0
    while True:
        x = b[i]; i += 1
        res |= (x & 0x7f) << shift
        if not (x & 0x80):
            return res, i
        shift += 7


def enc_varint(n):
    out = bytearray()
    while True:
        x = n & 0x7f; n >>= 7
        out.append(x | 0x80 if n else x)
        if not n:
            return bytes(out)


def parse(b):
    i = 0; out = []
    while i < len(b):
        tag, i = read_varint(b, i)
        fn, wt = tag >> 3, tag & 7
        if wt == 0:
            val, i = read_varint(b, i); out.append((fn, 0, val))
        elif wt == 2:
            ln, i = read_varint(b, i); out.append((fn, 2, b[i:i+ln])); i += ln
        else:
            raise Exception("unhandled wiretype %d" % wt)
    return out


def serialize(fields):
    out = bytearray()
    for fn, wt, payload in fields:
        out += enc_varint((fn << 3) | wt)
        out += enc_varint(payload) if wt == 0 else enc_varint(len(payload)) + payload
    return bytes(out)


con = sqlite3.connect(DB)
rows = con.execute("SELECT key, value FROM ProtobufItems ORDER BY rowid").fetchall()
items = {k: (v if isinstance(v, (bytes, bytearray)) else str(v).encode()) for k, v in rows}

target = None
for k, v in items.items():
    if k == "Checksum":
        continue
    try:
        for fn, wt, payload in parse(v):
            if fn == 17 and wt == 2:
                sub = parse(payload)
                if sub and sub[0][0] == 1 and b"V" in sub[0][2][:12]:
                    target = k
    except Exception:
        pass
if target is None:
    sys.exit("ERROR: identity item not found in ProtobufItems")

newtop = []
for fn, wt, payload in parse(items[target]):
    if fn == 17 and wt == 2:
        newsub = []
        for sfn, swt, spayload in parse(payload):
            if sfn == 1:
                newsub.append((1, 2, NEW_ID))
            elif sfn == 3:
                newsub.append((3, 2, NEW_NICK))
            else:
                newsub.append((sfn, swt, spayload))
        newtop.append((17, 2, serialize(newsub)))
    else:
        newtop.append((fn, wt, payload))
items[target] = serialize(newtop)

con.execute("UPDATE ProtobufItems SET value=? WHERE key=?", (sqlite3.Binary(items[target]), target))
order = sorted([k for k in items if k != "Checksum"], key=lambda x: int(x))
digest = hashlib.sha1(b"".join(items[k] for k in order)).digest()
con.execute("UPDATE ProtobufItems SET value=? WHERE key='Checksum'", (sqlite3.Binary(digest),))
con.commit()
print("Injected identity (item %s), recomputed checksum %s" % (target, digest.hex()))
