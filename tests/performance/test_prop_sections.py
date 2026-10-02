import hashlib
import importlib.util
import json
import tempfile
from pathlib import Path
import unittest
from urllib.parse import urlparse,parse_qs
from PIL import Image

ROOT=Path(__file__).resolve().parents[2]
spec=importlib.util.spec_from_file_location('prop_sections',ROOT/'scripts/build-brawl-prop-sections.py')
module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)

class PropSectionsTest(unittest.TestCase):
    def test_all_pixels_hashes_and_overlapping_reconstruction(self):
        root=ROOT/'internal/bot/webassets'
        manifest=json.loads((root/'rift_prop_sections.js').read_text(encoding='utf-8').split('window.RiftPropSections=',1)[1].strip().removesuffix(';'))
        raw=(root/'rift_props.png').read_bytes()
        self.assertEqual(manifest['sourceSHA256'],hashlib.sha256(raw).hexdigest())
        self.assertEqual(manifest['regions'],[0,1,2,3,4,5,6,3,3,7])
        self.assertEqual(len(manifest['panels']),8)
        with Image.open(root/'rift_props.png') as image:original=image.convert('RGBA')
        for order in [range(8),reversed(range(8))]:
            combined=Image.new('RGBA',original.size)
            for index in order:
                s=manifest['panels'][index];url=urlparse(s['url']);path=root/Path(url.path).name
                self.assertEqual(parse_qs(url.query),{'v':[hashlib.sha256(path.read_bytes()).hexdigest()[:12]]})
                with Image.open(path) as image:part=image.convert('RGBA')
                self.assertEqual(part.size,(s['width'],s['height']))
                box=(s['left'],s['top'],s['left']+s['width'],s['top']+s['height'])
                self.assertEqual(part.tobytes(),original.crop(box).tobytes())
                combined.paste(part,(s['left'],s['top']))
            self.assertEqual(combined.tobytes(),original.tobytes())

    def test_changed_layout_or_dimensions_requires_review(self):
        source=(ROOT/'internal/bot/webassets/rift_renderer.js').read_text(encoding='utf-8')
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory)
            changed=source.replace('index=[0,1,2,3,4,5,6,3,3,7]','index=[1,0,2,3,4,5,6,3,3,7]')
            self.assertNotEqual(changed,source)
            (root/'rift_renderer.js').write_text(changed,encoding='utf-8')
            with self.assertRaisesRegex(ValueError,'mapping'):module.artifacts(root)
            (root/'rift_renderer.js').write_text(source,encoding='utf-8')
            Image.new('RGBA',(16,8)).save(root/'rift_props.png')
            with self.assertRaisesRegex(ValueError,'dimensions'):module.artifacts(root)

if __name__=='__main__':unittest.main()
