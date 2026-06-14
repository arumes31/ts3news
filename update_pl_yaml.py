
import sys

target_file = 'internal/i18n/locales/pl_PL.yaml'
names_file = 'pl_names.txt'

# Read new translations
with open(names_file, 'r', encoding='utf-8') as f:
    new_names = f.read()

# Read target file and find where to replace
with open(target_file, 'r', encoding='utf-8') as f:
    lines = f.readlines()

output_lines = []
for line in lines:
    if line.startswith('pool.channel.name.0001:'):
        break
    output_lines.append(line)

# Append the new names
output_lines.append(new_names)

# Write back to target file
with open(target_file, 'w', encoding='utf-8') as f:
    f.writelines(output_lines)

print("Successfully updated pl_PL.yaml")
