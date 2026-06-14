
with open('ru_translations.txt', 'r', encoding='utf-8') as f:
    translations = f.read()

with open('internal/i18n/locales/ru_RU.yaml', 'r', encoding='utf-8') as f:
    lines = f.readlines()

new_lines = []
found_start = False
for line in lines:
    if line.startswith('pool.channel.name.0001:'):
        found_start = True
        new_lines.append(translations)
    elif found_start:
        if not line.startswith('pool.channel.name.'):
            found_start = False
            new_lines.append(line)
    else:
        new_lines.append(line)

with open('internal/i18n/locales/ru_RU.yaml', 'w', encoding='utf-8') as f:
    f.writelines(new_lines)
