# Brawl localization candidate inventory

Run from the repository root with Go installed, after `npm ci`:

```powershell
node scripts/brawl-string-inventory.cjs --out .tmp/brawl-localization-candidates.json
node --test scripts/brawl-string-inventory.test.cjs
go test ./cmd/brawl-string-source
```

The development-only Acorn and parse5 parsers read all
`internal/bot/webassets/rift*.js`, `rift*.html` and shared `abyss*.js` files.
The command also runs the standard-library-only Go source extractor at
`cmd/brawl-string-source`, covering non-test `internal/rift/*.go`,
`internal/content/*.go` and `internal/bot/web_rift*.go` files. It never executes those files. Output includes source-file SHA256 hashes,
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

HTML extraction includes text nodes, title/alt/placeholder and accessible text
attributes, plus button input values. Comments, scripts and styles are excluded.
Go template actions are masked before HTML parsing so nested quotes in asset
helpers cannot corrupt attribute parsing, then restored as numbered parameters.
Template control actions can appear among candidates and require source review;
do not translate them. Attribute locations point to the attribute start.

The scope includes Brawl JavaScript/HTML, Brawl server/engine source and shared
Abyss JavaScript/content source. Shared modules deliberately over-include strings
used by other Abyss views; retain ownership during review. Go raw and interpreted
string literals are decoded by the Go parser with one-based byte-column positions
(JavaScript/HTML columns use their parser's character positions). Imports, struct
tags, identifiers and format strings are candidates too. Go formatting verbs are
preserved verbatim, requiring format/argument review before translation.

This is a source inventory, not an enumeration of every possible generated
sentence. Runtime player/item names, database values, external service errors and
shared site navigation outside these patterns are not exported. Static message
fragments/formatters in the selected sources remain available for review. This tooling adds no runtime
locale selection, translations or browser bundle dependency.

Generated files belong in `.tmp` and are review artifacts. Keep accepted translator
work separately before regenerating: this command intentionally produces a fresh
inventory and does not merge translation edits. Record commit and source hashes
alongside reviewed output. The JSON contains application source copy, not accounts,
player data or runtime saves.
