import glob
from deep_translator import GoogleTranslator

# 1. Read canonical English strings
en_names = {}
with open('internal/i18n/locales/en_US.yaml', 'r', encoding='utf-8') as f:
    for line in f:
        if line.startswith('pool.channel.name.'):
            key, val = line.split(':', 1)
            key = key.strip()
            val = val.strip().strip('"')
            en_names[key] = val

locales = {
    'ar_SA': 'ar', 'cs_CZ': 'cs', 'de_DE': 'de', 'es_ES': 'es', 'fr_FR': 'fr', 
    'hi_IN': 'hi', 'it_IT': 'it', 'ja_JP': 'ja', 'ko_KR': 'ko', 'nl_NL': 'nl', 
    'pl_PL': 'pl', 'pt_BR': 'pt', 'ru_RU': 'ru', 'sv_SE': 'sv', 'th_TH': 'th', 
    'tr_TR': 'tr', 'vi_VN': 'vi', 'zh_CN': 'zh-CN', 'zh_TW': 'zh-TW'
}

for locale, lang_code in locales.items():
    filepath = f'internal/i18n/locales/{locale}.yaml'
    print(f"Processing {locale}...")
    
    with open(filepath, 'r', encoding='utf-8') as f:
        lines = f.readlines()
        
    translator = GoogleTranslator(source='en', target=lang_code)
    
    # Collect indices of lines that need translation
    to_translate_idx = []
    texts_to_translate = []
    
    for i, line in enumerate(lines):
        if line.startswith('pool.channel.name.'):
            key, val = line.split(':', 1)
            key = key.strip()
            val = val.strip().strip('"')
            
            # If the current translation equals the English original, it needs translation
            if val == en_names.get(key, ""):
                to_translate_idx.append(i)
                texts_to_translate.append(val)
                
    if not texts_to_translate:
        print(f"{locale} is fully translated.")
        continue
        
    print(f"Translating {len(texts_to_translate)} strings for {locale}...")
    
    translated_texts = []
    batch_size = 50
    for i in range(0, len(texts_to_translate), batch_size):
        batch = texts_to_translate[i:i+batch_size]
        try:
            res = translator.translate_batch(batch)
            translated_texts.extend(res)
            print(f"  Translated {i+len(batch)}/{len(texts_to_translate)}")
        except Exception as e:
            print(f"  Error at batch {i}: {e}. Falling back to original.")
            translated_texts.extend(batch)
            
    # Replace in file
    for idx, new_val in zip(to_translate_idx, translated_texts):
        # We need to preserve the key format
        key = lines[idx].split(':', 1)[0]
        # remove single quotes or double quotes inside the string to prevent yaml breakage
        safe_val = new_val.replace('"', "'")
        lines[idx] = f'{key}: "{safe_val}"\n'
        
    with open(filepath, 'w', encoding='utf-8') as f:
        f.writelines(lines)
    print(f"Updated {locale}.\n")
