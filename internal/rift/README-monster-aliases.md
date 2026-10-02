# Canonical monster alias search

Abyss `content.Mob.Aliases` holds optional authored alternate search names. Brawl
reads these from the live canonical mob catalog into public bestiary metadata.
There is no separate Brawl alias registry. Catalog clones and bestiary entries
copy the slice so callers cannot mutate the authored source through a response.

The displayed name, art key, records, bookmarks and encounter identity continue
to use the canonical name. Aliases only add normalized search terms, alongside
existing name, family, tier, element and attack-style terms. Existing filters
still apply. Search ignores malformed non-array alias data and non-string values;
alias text is assigned as text data, never interpreted as HTML. Metadata omits
aliases when none are supplied, preserving old-client and old-catalog behavior.

The current canonical catalog authors no aliases. No nicknames, translations or
names inferred from art rigs were added. Future aliases should be authored on
the canonical Mob template; they will flow to Brawl without another registration.
This implements improvement0712's conditional alias-search support.

Validation covers authored test-template propagation, deep-copy isolation,
canonical identity, omitted metadata, case/accent/punctuation normalization,
existing family filters, malformed alias values and HTML-like plain text.
