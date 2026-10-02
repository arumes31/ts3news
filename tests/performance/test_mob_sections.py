import hashlib
import importlib.util
import tempfile
import json
from pathlib import Path
import unittest
from urllib.parse import urlparse, parse_qs
from PIL import Image

spec = importlib.util.spec_from_file_location('mob_sections', Path(__file__).resolve().parents[2] / 'scripts/build-brawl-mob-sections.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class MobSectionsTest(unittest.TestCase):
    def test_generated_rows_preserve_every_rgba_pixel_and_hash(self):
        root = Path(__file__).resolve().parents[2] / 'internal/bot/webassets'
        text = (root / 'rift_mob_sections.js').read_text(encoding='utf-8')
        manifest = json.loads(text.split('window.RiftMobSections=', 1)[1].strip().removesuffix(';'))
        self.assertEqual(manifest['version'], 1)
        self.assertEqual([row['kind'] for row in manifest['rows']], ['goblin','archer','knight','boss','wolf','spore'])
        raw = (root / 'rift_mobs.png').read_bytes()
        self.assertEqual(manifest['sourceSHA256'], hashlib.sha256(raw).hexdigest())
        with Image.open(root / 'rift_mobs.png') as image:
            original = image.convert('RGBA')
        for index, row in enumerate(manifest['rows']):
            self.assertEqual((row['y'], row['width'], row['height']), (index*128, 2048, 128))
            url = urlparse(row['url'])
            path = root / Path(url.path).name
            self.assertEqual(parse_qs(url.query), {'v':[hashlib.sha256(path.read_bytes()).hexdigest()[:12]]})
            with Image.open(path) as image:
                self.assertEqual(image.size, (2048, 128))
                self.assertEqual(image.convert('RGBA').tobytes(), original.crop((0,index*128,2048,(index+1)*128)).tobytes())

    def test_changed_renderer_layout_and_atlas_dimensions_require_review(self):
        source = (Path(__file__).resolve().parents[2] / 'internal/bot/webassets/rift_renderer.js').read_text(encoding='utf-8')
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            changed = source.replace('goblin:0,archer:1', 'goblin:1,archer:0')
            self.assertNotEqual(changed, source)
            (root/'rift_renderer.js').write_text(changed, encoding='utf-8')
            with self.assertRaisesRegex(ValueError, 'row order'):
                module.artifacts(root)
            (root/'rift_renderer.js').write_text(source, encoding='utf-8')
            Image.new('RGBA',(16,12)).save(root/'rift_mobs.png')
            with self.assertRaisesRegex(ValueError, 'dimensions'):
                module.artifacts(root)

if __name__ == '__main__':
    unittest.main()
