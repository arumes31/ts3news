import sys

en_names = []
with open('internal/i18n/locales/en_US.yaml', 'r', encoding='utf-8') as f:
    for line in f:
        if line.startswith('pool.channel.name.'):
            parts = line.split(':')
            if len(parts) >= 2:
                name = parts[1].strip().strip('"')
                en_names.append(name)
en_names = en_names[:1000]

adjs = set()
nouns = set()
for name in en_names:
    parts = name.split(' ', 1)
    if len(parts) == 2:
        adjs.add(parts[0])
        nouns.add(parts[1])

print("Adjectives:", len(adjs))
print("Nouns:", len(nouns))
