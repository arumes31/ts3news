import re

with open('translated_block.txt', 'r', encoding='utf-8') as f:
    translated_lines = f.readlines()

translated_dict = {}
for line in translated_lines:
    if line.startswith('pool.channel.name.'):
        key, value = line.split(':', 1)
        translated_dict[key.strip()] = value.strip()

with open('internal/i18n/locales/nl_NL.yaml', 'r', encoding='utf-8') as f:
    nl_content = f.read()

for key, value in translated_dict.items():
    nl_content = re.sub(rf'{key}:.*', f'{key}: {value}', nl_content)

with open('internal/i18n/locales/nl_NL.yaml', 'w', encoding='utf-8') as f:
    f.write(nl_content)

print("nl_NL.yaml updated successfully.")
