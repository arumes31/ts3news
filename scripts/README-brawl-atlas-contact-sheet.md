# Brawl atlas contact sheet

Run from the repository root:

```powershell
node scripts/brawl-atlas-contact-sheet.cjs
```

Open `.tmp/brawl-atlas-contact-sheet.html` locally. An optional first argument
selects another output path (its parent directory must already exist). The sheet
uses local file URLs and requires this checkout; it is not a portable published
website. No game server or network service is required.

The inventory includes every `rift*.png` and seven shared Abyss combat/class
atlases used by Brawl. Each card links to the original image and shows dimensions,
file size and SHA-256. Search filters filenames; checkerboard, dark and light
backgrounds reveal transparency edges. Original files are never altered.

This is a full-sheet art review, not proof that every sprite crop or animation
frame is aligned. Use runtime animation and atlas-boundary checks for that.
Regenerate after artwork changes to update metadata and fingerprints.
