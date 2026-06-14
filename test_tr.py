import urllib.request, urllib.parse, json

with open('internal/i18n/locales/en_US.yaml', 'r', encoding='utf-8') as f:
    lines = [line.split(':', 1)[1].strip().strip('"') for line in f if 'pool.channel.name.' in line][:20]

text = '\n'.join(lines)
url = 'https://translate.googleapis.com/translate_a/single?client=gtx&sl=en&tl=cs&dt=t&q=' + urllib.parse.quote(text)
req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
try:
    res = urllib.request.urlopen(req).read().decode('utf-8')
    data = json.loads(res)
    out = ''.join([x[0] for x in data[0]])
    print(out)
except Exception as e:
    print('Error:', e)
