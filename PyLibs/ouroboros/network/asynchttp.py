from ..common import *
from ..otypes import *
import aiohttp
import aiohttp.web_exceptions
import asyncio
import http.cookies
import urllib

def canonicalHeaderKey(key):
    return '-'.join([s[0].upper() + s[1:].lower() for s in key.split('-')])

class _CaseInsensitiveDict(CaseInsensitiveDict):
    def add(self, key, value):
        self[key] = value

aiohttp.client.ClientRequest.DEFAULT_HEADERS = {}

class _ClientRequest(aiohttp.client.ClientRequest):
    def __init__(self, *args, noQuotoPath = False, **kwargs):
        self.noQuotoPath = noQuotoPath
        super().__init__(*args, **kwargs)

    @classmethod
    def factory(cls, **customArgs):
        def wrapper(*args, **kwargs):
            kwargs.update(customArgs)
            return cls(*args, **kwargs)

        return wrapper

    def update_path(self, params):
        """Build path."""
        # extract path
        scheme, netloc, path, query, fragment = urllib.parse.urlsplit(self.url)
        if not path:
            path = '/'

        if isinstance(params, dict):
            params = list(params.items())
        elif isinstance(params, (aiohttp.MultiDictProxy, aiohttp.MultiDict)):
            params = list(params.items())

        if params:
            params = urllib.parse.urlencode(params)
            if query:
                query = '%s&%s' % (query, params)
            else:
                query = params

        if not self.noQuotoPath:
            path = urllib.parse.quote(path, safe='/%:')

        self.path = urllib.parse.urlunsplit(('', '', path, query, fragment))

    def update_cookies(self, cookies):
        """Update request cookies header."""
        if not cookies:
            return

        c = http.cookies.BaseCookie()
        if aiohttp.hdrs.COOKIE in self.headers:
            c.load(self.headers.get(aiohttp.hdrs.COOKIE, ''))
            del self.headers[aiohttp.hdrs.COOKIE]

        if isinstance(cookies, dict):
            cookies = cookies.items()

        for name, value in cookies:
            if isinstance(value, http.cookies.Morsel):
                # use dict method because SimpleCookie class modifies value
                dict.__setitem__(c, name, value)
            else:
                c[name] = value

        self.headers['Cookie'] = c.output(header='', sep=';', attrs = {}).strip()

class AsyncHttp:
    class Response:
        def __init__(self, response, content):
            self.response = response
            self.content = content

        def __repr__(self):
            return self.response.__repr__()

        def __str__(self):
            return self.response.__str__()

        @property
        def status(self):
            return self.response.status

        def plist(self):
            import plistlib
            return plistlib.loads(self.content)

        def json(self, encoding = None):
            return dict2(json.loads(self.decode(encoding)))

        def text(self, encoding = None):
            return self.decode(encoding)

        def decode(self, encoding = None, **kwargs):
            return self.content.decode(encoding or self.response.get_encoding(), **kwargs)

    def __init__(self, *, loop = None, timeout = 30, cookie_class = http.cookies.BaseCookie, verify_ssl = True):
        self.loop = loop or asyncio.get_event_loop()

        self.headers = {}
        self.timeout = timeout
        self.cookie_jar = aiohttp.CookieJar(loop = self.loop)
        self.proxy = None
        self.proxyAuth = None

        self.session = aiohttp.ClientSession(
                            loop        = self.loop,
                            cookie_jar  = self.cookie_jar,
                            # version     = aiohttp.http.HttpVersion10,
                            connector   = aiohttp.TCPConnector(loop = self.loop, verify_ssl = verify_ssl),
                        )

    def __del__(self):
        self.close()

    def close(self):
        asyncio.ensure_future(self.session.close(), loop = self.loop)

    @property
    def cookies(self):
        return self.cookie_jar

    def SetHeaders(self, headers):
        self.headers = headers

    def AddHeaders(self, headers):
        self.headers.update(headers)

    def SetCookies(self, cookies):
        if not cookies:
            self.cookie_jar.clear()
        else:
            self.cookie_jar.update_cookies(cookies)

    def SetProxy(self, host, port, login = None, password = None, encoding = 'latin1'):
        self.proxy = 'http://%s:%s' % (host, port)

        if login:
            self.proxyAuth = aiohttp.BasicAuth(login, password, encoding)
        else:
            self.proxyAuth = None

    def ClearProxy(self):
        self.proxy = None
        self.proxyAuth = None

    async def get(self, url, **kwargs):
        return await self.request('get', url, **kwargs)

    async def post(self, url, **kwargs):
        allow_redirects = kwargs.setdefault('allow_redirects', False)

        method = 'post'

        while True:
            resp = await self.request(method, url, **kwargs)

            if resp.status in [aiohttp.web_exceptions.HTTPFound.status_code]:
                method = 'get'
                for p in ['data', 'headers']:
                    try:
                        del kwargs[p]
                    except KeyError:
                        pass

            if resp.status in [
                    aiohttp.web_exceptions.HTTPFound.status_code,
                    aiohttp.web_exceptions.HTTPMovedPermanently.status_code,
                    aiohttp.web_exceptions.HTTPSeeOther.status_code,
                    aiohttp.web_exceptions.HTTPTemporaryRedirect.status_code,
                ]:
                url = resp.response.headers.get(aiohttp.hdrs.LOCATION)
                continue

            break

        return resp

    async def request(self, method, url, **kwargs):
        params = {}

        for key in ['noQuotoPath']:
            try:
                params[key] = kwargs.pop(key)
            except KeyError:
                pass

        # kwargs['request_class'] = _ClientRequest.factory(**params)

        hdr = self.headers.copy()
        hdr.update(kwargs.get('headers', {}))
        kwargs['headers'] = hdr

        if self.proxy:
            kwargs['proxy'] = self.proxy

        if self.proxyAuth:
            kwargs['proxy_auth'] = self.proxyAuth

        import datetime

        try:
            response = await self.session.request(method, url, timeout = self.timeout, **kwargs)
            content = await asyncio.wait_for(response.read(), self.timeout)
        except asyncio.TimeoutError:
            raise asyncio.TimeoutError('%s %s timeout: %s' % (method, url, self.timeout))

        return self.Response(response, content)
