"""Measure lossless per-class transport sections without writing or changing art."""
import argparse
import hashlib
import io
import json
from pathlib import Path
import re

from PIL import Image, __version__ as pillow_version, features


def encode_section(image, format):
    output = io.BytesIO()
    options = {"lossless": True, "exact": True, "method": 6} if format == "WEBP" else {"optimize": True}
    image.save(output, format=format, **options)
    encoded = output.getvalue()
    with Image.open(io.BytesIO(encoded)) as decoded:
        if decoded.size != image.size or decoded.convert("RGBA").tobytes() != image.tobytes():
            raise ValueError("Section encoding changed pixels")
    return encoded


def split_rows(image, count):
    if count <= 0 or image.height % count:
        raise ValueError("Atlas must contain equally sized integer rows")
    height = image.height // count
    return [image.crop((0, row * height, image.width, (row + 1) * height)) for row in range(count)]


def hero_classes(root):
    source = (root / "rift_renderer.js").read_text(encoding="utf-8")
    match = re.search(r"const styles = \[(.*?)\]", source)
    if not match:
        raise ValueError("Renderer class layout was not found")
    classes = re.findall(r"'([^']+)'", match[1])
    if len(classes) != 12 or len(set(classes)) != 12:
        raise ValueError("Review the transport plan for the changed class layout")
    return classes


def measure(root):
    if not features.check("webp"):
        raise RuntimeError("Pillow must include WebP support")
    classes = hero_classes(root)
    sheets = []
    for sheet_index, name in enumerate(["rift_heroes_a.png", "rift_heroes_b.png"]):
        raw = (root / name).read_bytes()
        with Image.open(io.BytesIO(raw)) as opened:
            original = opened.convert("RGBA")
        rows = split_rows(original, 6)
        reconstructed = Image.new("RGBA", original.size)
        sections = []
        for index, row in enumerate(rows):
            # No alpha mask: paste copies RGBA bytes, including invisible RGB.
            reconstructed.paste(row, (0, index * row.height))
            png, webp = encode_section(row, "PNG"), encode_section(row, "WEBP")
            sections.append({"class": classes[sheet_index * 6 + index], "row": index,
                             "x": 0, "y": index * row.height, "width": row.width, "height": row.height,
                             "png_bytes": len(png), "webp_bytes": len(webp),
                             "rgba_sha256": hashlib.sha256(row.tobytes()).hexdigest()})
        if reconstructed.tobytes() != original.tobytes():
            raise ValueError("Reconstructed atlas changed pixels: " + name)
        sheets.append({"asset": name, "source_sha256": hashlib.sha256(raw).hexdigest(),
                       "source_bytes": len(raw), "width": original.width, "height": original.height,
                       "sections": sections, "reconstruction_pixel_exact": True})
    return {"schema": "brawl-hero-sections-v1", "pillow": pillow_version,
            "libwebp": features.version("webp"), "sheets": sheets,
            "scope": "In-memory encoding experiment only. Original artwork and production loading unchanged. "
                     "Browser compositing, filtering, animation, caching, and readiness remain unverified."}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    result = json.dumps(measure(Path(__file__).resolve().parents[1] / "internal/bot/webassets"), indent=2) + "\n"
    if args.output:
        args.output.write_text(result, encoding="utf-8")
    else:
        print(result, end="")
