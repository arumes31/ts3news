"""Read-only audit of every catalog icon at actual UI thumbnail sizes."""
import json
from collections import Counter
from hashlib import sha256
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT / "internal" / "bot" / "webassets"
raw = (ASSETS / "abyss_catalog_icons.js").read_text(encoding="utf-8")
catalog = json.loads(raw.split("=", 1)[1].strip().removesuffix(";"))
images = {}
seen = {size: {} for size in (96, 48, 32)}
for key, entry in catalog.items():
    filename = entry["asset"].removeprefix("/static/")
    if filename not in images:
        images[filename] = Image.open(ASSETS / filename).convert("RGBA")
        assert images[filename].size == (1344, 1152), filename
    x, y = entry["column"] * 96, entry["row"] * 96
    icon = images[filename].crop((x, y, x + 96, y + 96))
    assert icon.getchannel("A").getbbox(), f"Empty icon: {key}"
    assert icon.getpixel((0, 0))[3] == 0, f"Opaque tile: {key}"
    # Discard the former invisible watermark before checking uniqueness.
    for py in (47, 48):
        for px in range(46, 50):
            icon.putpixel((px, py), (0, 0, 0, 0))
    for size in seen:
        thumbnail = icon.resize((size, size), Image.Resampling.LANCZOS)
        signature = sha256(thumbnail.tobytes()).hexdigest()
        assert signature not in seen[size], f"Duplicate at {size}px: {key} / {seen[size].get(signature)}"
        seen[size][signature] = key

assert len(catalog) >= 2800
counts = Counter(entry["kind"] for entry in catalog.values())
print(json.dumps({"icons": len(catalog), "pages": len(images), "kinds": counts,
                  "unique_at_pixels": list(seen)}, indent=2))
