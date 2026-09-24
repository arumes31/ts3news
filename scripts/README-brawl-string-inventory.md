# Brawl localization candidate inventory

Run from the repository root after `npm ci`:

```powershell
node scripts/brawl-string-inventory.cjs --out .tmp/brawl-localization-candidates.json
node --test scripts/brawl-string-inventory.test.cjs
```

The development-only Acorn parser reads all `internal/bot/webassets/rift*.js`
files. It never executes those files. Output includes source-file SHA256 hashes,
string text, one-based line/column locations, parent AST context and all duplicate
occurrences. IDs hash string kind and exact text, remaining stable when a line
moves. Source edits change the relevant file hash; regenerate after changing copy.
Syntax errors fail explicitly rather than silently omitting part of a file.

Every entry starts with `translation: null` and `review: unreviewed`. This is a
candidate inventory, not a list of approved translations. It deliberately includes
identifiers, selectors, URLs, fragments and diagnostic text so extraction does not
silently guess away potentially visible strings. Review AST context and source
before deciding whether to translate, exclude or combine fragments into a message.
Do not feed the entire output directly into runtime localization.

Template literals use numbered `{1}` placeholders and retain the exact expression
source per occurrence. These expressions are context only; never execute them as
translation code. Ordinary string concatenations remain separate candidates and
need human review to define complete messages, plural rules and parameter types.
English text in the core `statusCopy` catalog is included automatically.

The initial scope is all Brawl JavaScript modules. HTML templates, Go-generated
mission/objective/loot text, shared Abyss content and dynamically received server
messages are not yet included. The full localization-inventory task remains open
until these sources and their ownership are covered. This tooling adds no runtime
locale selection, translations or browser bundle dependency.

Generated files belong in `.tmp` and are review artifacts. Keep accepted translator
work separately before regenerating: this command intentionally produces a fresh
inventory and does not merge translation edits. Record commit and source hashes
alongside reviewed output. The JSON contains application source copy, not accounts,
player data or runtime saves.
