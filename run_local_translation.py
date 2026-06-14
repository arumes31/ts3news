import urllib.request
import json
import re
import os
import time

def translate_batch(texts, target_lang):
    # Free Google Translate API endpoint (undocumented, used for simple batch tasks)
    results = []
    print(f"Translating {len(texts)} items. Progress: ", end="", flush=True)
    for i, text in enumerate(texts):
        if i % 100 == 0:
            print(f"{i}..", end="", flush=True)
            
        url = f"https://translate.googleapis.com/translate_a/single?client=gtx&sl=en&tl={target_lang}&dt=t&q={urllib.parse.quote(text)}"
        try:
            req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
            with urllib.request.urlopen(req) as response:
                res = json.loads(response.read().decode('utf-8'))
                results.append(res[0][0][0].replace('"', "'")) # prevent breaking YAML
            time.sleep(0.05) # Be gentle to the free API
        except Exception as e:
            print(f"E", end="")
            results.append(text) # Fallback to english
    print("Done")
    return results

def process_locale(locale_code, target_lang):
    print(f"Processing {locale_code}...")
    filepath = f"internal/i18n/locales/{locale_code}.yaml"
    
    with open(filepath, 'r', encoding='utf-8') as f:
        lines = f.readlines()
        
    start_idx = -1
    for i, line in enumerate(lines):
        if line.startswith('pool.channel.name.0001:'):
            start_idx = i
            break
            
    if start_idx == -1:
        print(f"Could not find start of channel names in {locale_code}")
        return
        
    # Extract English texts
    keys = []
    texts = []
    for i in range(start_idx, min(start_idx + 1000, len(lines))):
        if line := lines[i].strip():
            match = re.match(r'(pool\.channel\.name\.\d+):\s*"(.*)"', line)
            if match:
                keys.append(match.group(1))
                texts.append(match.group(2))
                
    if len(texts) < 1000:
        print(f"Warning: Only found {len(texts)} keys in {locale_code}")
        
    translated = translate_batch(texts, target_lang)
    
    # Replace in file
    for i in range(len(keys)):
        lines[start_idx + i] = f'{keys[i]}: "{translated[i]}"\n'
        
    with open(filepath, 'w', encoding='utf-8') as f:
        f.writelines(lines)
    print(f"Finished {locale_code}\n")

locales = {
    'ar_SA': 'ar',
    'cs_CZ': 'cs',
    'de_DE': 'de',
    'hi_IN': 'hi',
    'ja_JP': 'ja',
    'ko_KR': 'ko',
    'pt_BR': 'pt',
    'sv_SE': 'sv',
    'th_TH': 'th',
    'tr_TR': 'tr',
    'vi_VN': 'vi',
    'zh_CN': 'zh-CN',
    'zh_TW': 'zh-TW'
}

for locale, lang in locales.items():
    process_locale(locale, lang)
