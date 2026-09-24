"""Measure pixel-exact WebP candidates in memory; never rewrite source artwork."""
import argparse
import hashlib
import io
import json
from pathlib import Path
import re
import sys
import time

from PIL import Image, __version__ as pillow_version, features


def required_art(root):
    html = (root / "rift.html").read_text(encoding="utf-8")
    renderer = (root / "rift_renderer.js").read_text(encoding="utf-8")
    art = (root / "abyss_combat_art.js").read_text(encoding="utf-8")
    keys = re.search(r"const criticalAtlasKeys = \[(.*?)\]", renderer).group(1)
    files = []
    for key in re.findall(r"'([^']+)'", keys):
        attribute = "data-" + re.sub(r"[A-Z]", lambda m: "-" + m[0].lower(), key)
        if key == "props":
            tag = re.search(r'<link[^>]+id="rift-props-asset"[^>]*>', html).group(0)
        else:
            tag = re.search(attribute + r'="(.*?)"}}"', html).group(0)
        files.append(re.search(r"/static/([\w.-]+)", tag).group(1))
    catalog = re.search(r"var atlasAssets = \[(.*?)\]", art, re.S).group(1)
    files.extend(re.findall(r"/static/([\w.-]+)", catalog))
    return list(dict.fromkeys(files))


def measure(root):
    if not features.check("webp"):
        raise RuntimeError("Pillow must include WebP support")
    rows = []
    for name in required_art(root):
        raw = (root / name).read_bytes()
        source = Image.open(io.BytesIO(raw)).convert("RGBA")
        encoded = io.BytesIO()
        started = time.perf_counter()
        source.save(encoded, format="WEBP", lossless=True, method=6, exact=True)
        candidate = encoded.getvalue()
        decoded = Image.open(io.BytesIO(candidate)).convert("RGBA")
        if source.size != decoded.size or source.tobytes() != decoded.tobytes():
            raise RuntimeError("Pixel mismatch: " + name)
        row = {"asset": name, "source_sha256": hashlib.sha256(raw).hexdigest(),
               "width": source.width, "height": source.height, "source_bytes": len(raw),
               "lossless_webp_bytes": len(candidate), "pixel_exact": True,
               "encode_seconds": round(time.perf_counter() - started, 3)}
        rows.append(row)
        print(name + ": " + str(len(raw)) + " -> " + str(len(candidate)), file=sys.stderr, flush=True)
    before = sum(row["source_bytes"] for row in rows)
    after = sum(row["lossless_webp_bytes"] for row in rows)
    return {"pillow": pillow_version, "libwebp": features.version("webp"),
            "assets": rows, "source_bytes": before, "lossless_webp_bytes": after,
            "saved_bytes": before - after, "saved_percent": round((before-after)*100/before, 2),
            "art_only_transfer_floor_seconds_at_200000_Bps": round(after/200000, 2),
            "art_only_fits_3000000_byte_gate": after <= 3000000,
            "note": "Encoding experiment, not a browser startup measurement. Excludes HTML, CSS, JS, API data, headers, latency, decoding and rendering."}


if __name__ == "__main__":
    root = Path(__file__).resolve().parents[1] / "internal/bot/webassets"
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, help="Write the JSON report as UTF-8")
    args = parser.parse_args()
    report = json.dumps(measure(root), indent=2) + "\n"
    if args.output:
        args.output.write_text(report, encoding="utf-8")
    else:
        print(report, end="")
