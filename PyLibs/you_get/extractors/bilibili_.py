#!/usr/bin/env python

from ..common import *
from ..extractor import VideoExtractor

class Bilibili(VideoExtractor):
    name = "Bilibili"

    # Bilibili media encoding options, in descending quality order.
    stream_types = [
        {'itag': '38', 'container': 'MP4', 'video_resolution': '3072p', 'video_encoding': 'H.264', 'video_profile': 'High', 'video_bitrate': '3.5-5', 'audio_encoding': 'AAC', 'audio_bitrate': '192'},
    ]

    def prepare(self, **kwargs):
        # scrape the html
        content = get_content(self.url)

        print(self.url)
        # extract title
        #self.title = match1(content,
        #                    r'<meta property="og:description" name="og:description" content="([^"]+)"')

        # extract raw urls
        #orig_img = match1(content,
        #                 r'<meta itemprop="image" content="([^"]+/originals/[^"]+)"')
        #twit_img = match1(content,
        #                  r'<meta property="twitter:image:src" name="twitter:image:src" content="([^"]+)"')

        # construct available streams
        #if orig_img: self.streams['original'] = {'url': orig_img}
        #if twit_img: self.streams['small'] = {'url': twit_img}

    def extract(self, **kwargs):
        for i in self.streams:
            # for each available stream
            s = self.streams[i]
            # fill in 'container' field and 'size' field (optional)
            _, s['container'], s['size'] = url_info(s['url'])
            # 'src' field is a list of processed urls for direct downloading
            # usually derived from 'url'
            s['src'] = [s['url']]

site = Bilibili()
download = site.download_by_url
# TBD: implement download_playlist

bilibili_download = download
