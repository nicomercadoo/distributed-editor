import json, urllib.request
try:
    with open('toxiproxy.json') as f:
        data = json.load(f)
    for proxy in data:
        for toxic in proxy.get('toxics', []):
            url = 'http://localhost:8474/proxies/' + proxy['name'] + '/toxics'
            req = urllib.request.Request(url, data=json.dumps(toxic).encode('utf-8'), headers={'Content-Type': 'application/json'})
            urllib.request.urlopen(req)
    print('    [+] Toxics inicializados correctamente')
except Exception as e:
    print('    [-] Aviso: No se pudieron cargar los toxics:', e)
