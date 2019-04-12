from ml import *

sess = network.AsyncHttp()
appToken = '58781ae52f6b4cb7a70da7ffb1163f30cb3d0a68cb951677dded025d1b3bfea9'

def signParams(data):
    s = '|'.join([
        'os:%s'         % data['os'],
        'version:%s'    % data['version'],
        'action:%s'     % data['action'],
        'time:%s'       % data['time'],
        'appToken:%s'   % appToken,
        'privateKey:e1be6b4cf4021b3d181170d1879a530a9e4130b69032144d5568abfd6cd6c1c2',
        'data:%s&'      % '&'.join(['%s=%s' % (k, data['data'][k]) for k in sorted(data['data'])]),
    ])

    return encoding.md5(s.encode('UTF8')).hex()

async def callapi(action, data):
    url = f'https://mob.mddcloud.com.cn/{action}'

    body = {
        'action'        : '/' + action,
        'os'            : 'iOS',
        'channel'       : 'AppStore',
        'time'          : str(int(time.time() * 1000)),
        'deviceNum'     : '8c57ea62ab694114812d3fa5d6e9753a',
        'deviceType'    : 1,
        'appToken'      : '58781ae52f6b4cb7a70da7ffb1163f30cb3d0a68cb951677dded025d1b3bfea9',
        'data'          : data,
        'version'       : '3.3.2',
    }

    body['sign'] = signParams(body)

    resp = await sess.post(
        url,
        data = json.dumps(body),
    )

    return resp.json()['data']

async def listVodSactions(vod):
    return await callapi(
        'api/vod/listVodSactions.action',
        data = {
            'vodUuid'           : vod,
            'rows'              : 100,
            'hasIntroduction'   : 1,
            'startRow'          : 0
        },
    )

async def getSection(section):
    return await callapi(
        'api/vod/getSaction.action',
        data = {
            'sactionUuid' : section,
        },
    )

async def search(keyword):
    resp = await callapi(
        'searchApi/search/getAllSearchResult332.action',
        data = {
            'keyWord' : keyword,
        },
    )

    for d in resp:
        if d['type'] != 3:
            continue

        vod = d['vodList'][0]
        if vod['name'] == keyword:
            print(vod)

        return vod['uuid']

    return None

@asynclib.main
async def run():
    sess.SetProxy('localhost', 6789)
    sess.SetHeaders({
        'Content-Type'      : 'application/json',
        'Accept-Encoding'   : 'gzip, deflate',
        'Accept'            : 'application/json',
        'User-Agent'        : 'Mdd/3.3.2 (ios 10.1.1)',
        'Accept-Language'   : 'zh-Hans-CN;q=1, en-CN;q=0.9, zh-Hant-CN;q=0.8, en-GB;q=0.7',
        'Referer'           : 'mdd',
    })

    vod = await search('九五至尊')
    if not vod:
        return

    resp = await listVodSactions(vod)

    for i, vod in enumerate(resp):
        sec = await getSection(vod['uuid'])
        print('%02d\n%s\n' % (i + 1, sec['oriUrl']))

        # break

def main():
    run()

    if sys.platform == 'win32':
        console.pause('done')

if __name__ == '__main__':
    Try(main)
