import sys
import yaml
from deep_translator import GoogleTranslator

# 1. Read en_US.yaml
en_names = []
with open('internal/i18n/locales/en_US.yaml', 'r', encoding='utf-8') as f:
    for line in f:
        if line.startswith('pool.channel.name.'):
            parts = line.split(':', 1)
            name = parts[1].strip().strip('"')
            en_names.append(name)
en_names = en_names[:1000]

# 2. Map adjectives manually for better RPG tone
adj_map = {
    "Screaming": "Skrikande",
    "Blessed": "Välsignad",
    "Mystic": "Mystisk",
    "Dark": "Mörk",
    "Cursed": "Förbannad",
    "Silver": "Silvrig",
    "Divine": "Gudomlig",
    "Sacred": "Helig",
    "Forbidden": "Förbjuden",
    "Hidden": "Dold",
    "New": "Ny",
    "Celestial": "Himmelsk",
    "Wild": "Vild",
    "Golden": "Gyllene",
    "Steel": "Stål-",
    "Deadly": "Dödlig",
    "Ancient": "Uråldrig",
    "Frozen": "Frusen",
    "Emerald": "Smaragdgrön",
    "Holy": "Helig",
    "Elder": "Äldre",
    "Lethal": "Livsfarlig",
    "Bright": "Strålande",
    "Void": "Tomhetens",
    "Iron": "Järn-",
    "Burning": "Brinnande",
    "Shadow": "Skuggornas",
    "Whispering": "Viskande",
    "Lost": "Förlorad",
    "Forgotten": "Glömd",
    "Cold": "Kall",
    "Abyssal": "Avgrunds-",
    "Silent": "Tyst",
    "Ethereal": "Eterisk",
    "Infernal": "Infernalisk",
    "Primal": "Ursprunglig",
    "Young": "Ung",
    "Defiled": "Skändad",
    "Hot": "Glödande",
    "Rusty": "Rostig"
}

nouns = []
for name in en_names:
    parts = name.split(' ', 1)
    if len(parts) == 2:
        nouns.append(parts[1])
    else:
        nouns.append(name) # fallback

# Remove duplicates to save translation requests
unique_nouns = list(set(nouns))

print(f"Translating {len(unique_nouns)} unique nouns...")
sv_nouns = []
batch_size = 50
translator = GoogleTranslator(source='en', target='sv')

for i in range(0, len(unique_nouns), batch_size):
    batch = unique_nouns[i:i+batch_size]
    try:
        translated_batch = translator.translate_batch(batch)
        sv_nouns.extend(translated_batch)
        print(f"Translated {i+len(batch)}/{len(unique_nouns)}")
    except Exception as e:
        print(f"Error at batch {i}: {e}")
        # fallback to English if error
        sv_nouns.extend(batch)

noun_map = dict(zip(unique_nouns, sv_nouns))

# 3. Create full names
translated_lines = []
for i, name in enumerate(en_names):
    parts = name.split(' ', 1)
    if len(parts) == 2:
        adj = parts[0]
        noun = parts[1]
        
        sv_adj = adj_map.get(adj, adj)
        sv_noun = noun_map.get(noun, noun).capitalize()
        
        if sv_adj.endswith('-'):
            full_name = f"{sv_adj[:-1]}{sv_noun.lower()}"
        else:
            full_name = f"{sv_adj} {sv_noun}"
    else:
        full_name = noun_map.get(name, name).capitalize()
        
    translated_lines.append(f'pool.channel.name.{i+1:04d}: "{full_name}"\n')

# 4. Modify sv_SE.yaml
with open('internal/i18n/locales/sv_SE.yaml', 'r', encoding='utf-8') as f:
    lines = f.readlines()

new_lines = []
for line in lines:
    if line.startswith('pool.channel.name.'):
        continue
    new_lines.append(line)

new_lines.extend(translated_lines)

with open('internal/i18n/locales/sv_SE.yaml', 'w', encoding='utf-8') as f:
    f.writelines(new_lines)

print("Updated sv_SE.yaml successfully.")
