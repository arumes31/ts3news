
import sys

with open('en_names.txt', 'r', encoding='utf-8') as f:
    words = []
    for line in f:
        if ': "' in line:
            name = line.split(': "')[1].strip('"\n')
            parts = name.split()
            if len(parts) >= 1:
                words.append(parts[-1])

unique_words = sorted(set(words))
with open('suffixes.txt', 'w', encoding='utf-8') as f:
    for w in unique_words:
        f.write(w + '\n')
