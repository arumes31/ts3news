import hashlib
import importlib.util
import json
from pathlib import Path
import unittest
from urllib.parse import urlparse, parse_qs
from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('region_sections', ROOT / 'scripts/build-brawl-region-sections.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class RegionSectionsTest(unittest.TestCase):
    def test_panel_pixels_hashes_and_fractional_sampling(self):
        root = ROOT / 'internal/bot/webassets'
        text = (root / 'rift_region_sections.js').read_text()
        manifest = json.loads(text.split('window.RiftRegionSections=', 1)[1].strip().removesuffix(';'))
        source = root / 'rift_regions.png'
        self.assertEqual(manifest['sourceSHA256'], hashlib.sha256(source.read_bytes()).hexdigest())
        with Image.open(source) as image:
            original = image.convert('RGBA')
        rows = [0, .179, .363, .559, .755, 1]
        self.assertEqual(len(manifest['regions']), 10)
        for index, section in enumerate(manifest['regions']):
            url = urlparse(section['url'])
            encoded = root / Path(url.path).name
            self.assertEqual(parse_qs(url.query), {'v': [hashlib.sha256(encoded.read_bytes()).hexdigest()[:12]]})
            x, y = section['x'], section['y']
            width, height = section['width'], section['height']
            with Image.open(encoded) as image:
                self.assertEqual(image.size, (width, height))
                self.assertEqual(image.convert('RGBA').tobytes(), original.crop((x, y, x+width, y+height)).tobytes())
            sx, sy, sw, sh = section['source']
            self.assertAlmostEqual(sx+x, index%2*529+2)
            self.assertAlmostEqual(sy+y, rows[index//2]*1487+2)
            self.assertEqual(sw, 525)
            self.assertAlmostEqual(sh, (rows[index//2+1]-rows[index//2])*1487-4)
            self.assertGreaterEqual(sx, 0)
            self.assertGreaterEqual(sy, 0)
            self.assertLessEqual(sx+sw, width)
            self.assertLessEqual(sy+sh, height)

    def test_rejects_invalid_panel_layout(self):
        for rows in ([0, .5, 1], [0, .179, .363, .559, 1, .755], [0, .179, .363, .559, .755, .9]):
            with self.assertRaises(ValueError):
                module.panel_geometry((1058,1487), rows)
        with self.assertRaises(ValueError):
            module.panel_geometry((1057,1487), [0,.179,.363,.559,.755,1])

if __name__ == '__main__':
    unittest.main()
