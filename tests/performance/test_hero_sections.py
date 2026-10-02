import importlib.util
import hashlib
import json
from pathlib import Path
import unittest
from urllib.parse import urlparse, parse_qs

from PIL import Image

spec = importlib.util.spec_from_file_location("hero_sections", Path(__file__).resolve().parents[2] / "scripts/measure-brawl-hero-sections.py")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class HeroSectionsTest(unittest.TestCase):
    def test_generated_manifest_hashes_geometry_and_pixels_match_sources(self):
        root = Path(__file__).resolve().parents[2] / "internal/bot/webassets"
        text = (root / "rift_hero_sections.js").read_text(encoding="utf-8")
        manifest = json.loads(text.split("window.RiftHeroSections=", 1)[1].strip().removesuffix(";"))
        self.assertEqual(manifest["version"], 1)
        self.assertEqual(set(manifest["atlases"]), {"heroesA", "heroesB"})
        for key, atlas in manifest["atlases"].items():
            source = root / ("rift_heroes_" + key[-1].lower() + ".png")
            self.assertEqual(hashlib.sha256(source.read_bytes()).hexdigest(), atlas["sourceSHA256"])
            with Image.open(source) as image:
                original = image.convert("RGBA")
            self.assertEqual((atlas["width"], atlas["height"]), original.size)
            self.assertEqual(len(atlas["rows"]), 6)
            for index, row in enumerate(atlas["rows"]):
                url = urlparse(row["url"])
                section = root / Path(url.path).name
                self.assertEqual(parse_qs(url.query), {"v": [hashlib.sha256(section.read_bytes()).hexdigest()[:12]]})
                self.assertEqual((row["y"], row["width"], row["height"]), (index * 128, 2048, 128))
                with Image.open(section) as image:
                    self.assertEqual(image.convert("RGBA").tobytes(), original.crop((0, index * 128, 2048, (index + 1) * 128)).tobytes())

    def test_rows_keep_transparent_rgb_and_order(self):
        image = Image.new("RGBA", (3, 6))
        pixels = [(x * 51, y * 37, 99, 0 if y % 2 else 127) for y in range(6) for x in range(3)]
        image.putdata(pixels)
        rows = module.split_rows(image, 3)
        self.assertEqual(b"".join(row.tobytes() for row in rows), image.tobytes())
        for row in rows:
            for format in ("PNG", "WEBP"):
                self.assertGreater(len(module.encode_section(row, format)), 0)

    def test_invalid_row_layout_does_not_drop_pixels(self):
        image = Image.new("RGBA", (3, 7))
        for count in (0, -1, 3, 8):
            with self.assertRaises(ValueError):
                module.split_rows(image, count)


if __name__ == "__main__":
    unittest.main()
