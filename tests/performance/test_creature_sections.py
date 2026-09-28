import hashlib
import importlib.util
import json
from pathlib import Path
import unittest
from urllib.parse import urlparse,parse_qs
from PIL import Image
ROOT=Path(__file__).resolve().parents[2]
spec=importlib.util.spec_from_file_location('creature_sections',ROOT/'scripts/build-brawl-creature-sections.py')
module=importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class CreatureSectionsTest(unittest.TestCase):
    def test_rows_preserve_rgba_and_include_sampling_border(self):
        root=ROOT/'internal/bot/webassets'
        text=(root/'rift_creature_sections.js').read_text(encoding='utf-8')
        manifest=json.loads(text.split('window.RiftCreatureSections=',1)[1].strip().removesuffix(';'))
        self.assertEqual(manifest['version'],1)
        self.assertEqual(len(manifest['rigs']),32)
        seen=set()
        for rig,panel in manifest['rigs'].items():
            source=root/Path(panel['sourceAsset']).name
            with Image.open(source) as opened:
                original=opened.convert('RGBA')
            self.assertEqual(manifest['sources'][panel['sourceAsset']]['sha256'],hashlib.sha256(source.read_bytes()).hexdigest())
            url=urlparse(panel['url']);file=root/Path(url.path).name
            self.assertEqual(parse_qs(url.query),{'v':[hashlib.sha256(file.read_bytes()).hexdigest()[:12]]})
            top,end=panel['top'],panel['top']+panel['height']
            self.assertEqual(top,max(0,panel['rowY']-1))
            self.assertEqual(end,min(1254,panel['rowY']+panel['rowHeight']+1))
            with Image.open(file) as opened:
                self.assertEqual(opened.size,(1254,panel['height']))
                self.assertEqual(opened.convert('RGBA').tobytes(),original.crop((0,top,1254,end)).tobytes())
            seen.add((panel['sourceAsset'],panel['rowY']))
        self.assertEqual(len(seen),32)

    def test_layout_validation_rejects_dropped_or_overlapping_rows(self):
        for rows in ([0,158,318,476,638,783,924,1086,1253], [0,158,318,476,638,638,924,1086,1254], [0,158,318]):
            with self.assertRaises(ValueError):module.validate_rows(rows)

if __name__=='__main__':unittest.main()
