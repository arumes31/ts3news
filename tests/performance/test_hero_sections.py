import importlib.util
from pathlib import Path
import unittest

from PIL import Image

spec = importlib.util.spec_from_file_location("hero_sections", Path(__file__).resolve().parents[2] / "scripts/measure-brawl-hero-sections.py")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class HeroSectionsTest(unittest.TestCase):
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
