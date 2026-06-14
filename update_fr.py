import io

with io.open('internal/i18n/locales/fr_FR.yaml', 'r', encoding='utf-8') as f:
    lines = f.readlines()

# Line 1497 is index 1496
start_index = 1496
if lines[start_index].startswith('pool.channel.name.0001:'):
    new_lines = lines[:start_index]
    with io.open('fr_names.txt', 'r', encoding='utf-8') as f2:
        translated_lines = f2.readlines()
    
    # Ensure they end with newline
    translated_lines = [l if l.endswith('\n') else l + '\n' for l in translated_lines]
    
    new_lines.extend(translated_lines)
    
    with io.open('internal/i18n/locales/fr_FR.yaml', 'w', encoding='utf-8') as f3:
        f3.writelines(new_lines)
    print("Successfully updated fr_FR.yaml")
else:
    print(f"Error: Line 1497 does not start with pool.channel.name.0001. It is: {lines[start_index]}")
