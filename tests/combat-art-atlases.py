"""Read-only validation of the original generated combat PNG atlases."""
from hashlib import sha256
from pathlib import Path

from PIL import Image

ASSETS = Path(__file__).resolve().parents[1] / "internal" / "bot" / "webassets"
ROWS = {
    "roles": [0, 158, 318, 476, 638, 783, 924, 1086, 1254],
    "creatures": [0, 144, 298, 441, 617, 789, 947, 1076, 1254],
    "bestiary": [0, 143, 281, 435, 591, 758, 900, 1056, 1254],
    "bosses": [0, 155, 312, 466, 625, 786, 941, 1085, 1254],
}

seen_frames = set()
total_bytes = 0
for atlas, rows in ROWS.items():
    filename = ASSETS / f"abyss_combat_{atlas}_v2.png"
    total_bytes += filename.stat().st_size
    with Image.open(filename) as image:
        assert image.mode == "RGBA", f"{atlas}: expected genuine alpha"
        assert image.size == (1254, 1254), f"{atlas}: crop metadata must match source"
        assert image.getchannel("A").getextrema() == (0, 255)
        for row in range(8):
            for column in range(8):
                cell = image.crop((round(column * 1254 / 8), rows[row],
                                   round((column + 1) * 1254 / 8), rows[row + 1]))
                alpha = cell.getchannel("A")
                occupied = sum(value > 80 for value in alpha.get_flattened_data())
                assert occupied > 300, f"{atlas} {row}:{column}: empty pose"
                assert occupied < cell.width * cell.height * 0.9, f"{atlas}: boxed background"
                signature = sha256(cell.tobytes()).hexdigest()
                assert signature not in seen_frames, f"{atlas} {row}:{column}: duplicate pose"
                seen_frames.add(signature)

assert len(seen_frames) == 256
assert total_bytes < 14_000_000
print(f"Validated {len(seen_frames)} distinct transparent pose cells in {len(ROWS)} atlases; {total_bytes:,} bytes.")

class_frames = set()
class_bytes = 0
for name, size in {
    "abyss_player_classes_v1.png": (1448, 1086),
    "abyss_subclasses_martial_v1.png": (1448, 1086),
    "abyss_subclasses_mystic_v1.png": (1536, 1024),
}.items():
    filename = ASSETS / name
    class_bytes += filename.stat().st_size
    with Image.open(filename) as image:
        assert image.mode == "RGBA" and image.size == size, name
        lo, hi = image.getchannel("A").getextrema()
        assert lo == 0 and hi >= 250, name
        width, height = image.size
        for row in range(6):
            for column in range(8):
                cell = image.crop((round(column * width / 8), round(row * height / 6),
                                   round((column + 1) * width / 8), round((row + 1) * height / 6)))
                alpha = list(cell.getchannel("A").get_flattened_data())
                assert sum(value > 80 for value in alpha) > 300, f"{name}: empty pose"
                # Large release glows may fill most of a cell; require actual transparent
                # pixels and a visually opaque actor (generated alpha may peak at 252), without imposing an arbitrary glow radius.
                assert min(alpha) == 0 and max(alpha) >= 250, f"{name}: missing genuine alpha"
                signature = sha256(cell.tobytes()).hexdigest()
                assert signature not in class_frames, f"{name}: duplicate pose"
                class_frames.add(signature)
assert len(class_frames) == 144
assert class_bytes < 10_000_000
print(f"Validated 144 additional class/subclass pose cells in 3 original atlases; {class_bytes:,} bytes.")
