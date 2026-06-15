import glob, re
count = 0
for f in glob.glob('internal/i18n/locales/*.yaml'):
    if 'en_US' in f: continue
    with open(f, 'r', encoding='utf-8') as file:
        for line in file:
            if line.startswith('pool.channel.name.'):
                val = line.split(':', 1)[1].strip().strip('\"')
                if re.match(r'^[A-Za-z ]+$', val):
                    count += 1
print('Untranslated strings:', count)
