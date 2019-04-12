package downloader

import (
    . "ml/strings"
    . "ml/dict"

    "fmt"
    "time"
    "regexp"
    "crypto/des"

    "spew"

    "ml/net/http"
    "ml/encoding/json"
    "ml/encoding/base64"
)

var youkuVideoIdPattern = regexp.MustCompile(`(?U)currentEncodeVid\s*:\s*'(.*)'`)
var youkuVideoIdPattern2 = regexp.MustCompile(`(?U)videoId2\s*=\s*'(.*)';`)

type YoukuVideoInfoSeg struct {
    totalMillisecondsAudio      int64
    totalMillisecondsVideo      int64
    size                        int64
    cdn_url                     String
    fileid                      String
    key                         String
    hd                          int
    container                   String
}

type YoukuVideoInfo struct {
    segs    []YoukuVideoInfoSeg

    //security struct {
    //    sid             String
    //    token           String
    //    encryptString   String
    //    ip              int64
    //}
}

type YoukuDownloader struct {
    *baseDownloader
    vid         String
    videoInfo   YoukuVideoInfo
}

func NewYouku(url String) Downloader {
    d := &YoukuDownloader{
        baseDownloader: newBase(url),
    }

    return d
}

func (self *YoukuDownloader) Analysis() (result AnalysisResult, err error) {
    result = AnalysisNotSupported

    resp, err := self.session.Get(self.url)
    if err != nil {
        return
    }

     content := resp.Text()
     m := youkuVideoIdPattern.FindStringSubmatch(content.String())
     if len(m) == 0 {
         m = youkuVideoIdPattern2.FindStringSubmatch(content.String())
     }

    vid := String(m[1])

    fmt.Println("vid", vid)

    err = self.getVideoInfo(vid)
    if err != nil {
        return
    }

    self.parseVideoInfo()

    return AnalysisSuccess, nil
}

//
// http://k.youku.com/player/getFlvPath/sid/1474526152477 10f8 1137_00/st/mp4/fileid/030008010057E0B89D83C2019C3C1CAEE308CE-FEF5-6CE1-579C-51C872568410
// start=0
// K=dbcbe7f68e9006d7282bbfbe
// hd=1
// myp=0
// ts=68
// ymovie=1
// ypp=0
// ctype=10
// ev=1
// token=0939
// oip=244858955
// ep=p6F36Pjts%2BtCLWpTKCmthZj6RxeWJBP6lq77e%2FGGIqfo1mmyw7J4c2kYmVCZ5DuJFtEHjVM2G9unL9LD4GYQw0Hi%2BuI2jYE4rOI3l9%2BAxO8c8VxcnEnSiNdLuaWLAtdsasw%2FFBjp2uM%3D
// p6F36Pjts+tCLWpTKCmthZj6RxeWJBP6lq77e/GGIqfo1mmyw7J4c2kYmVCZ5DuJFtEHjVM2G9unL9LD4GYQw0Hi+uI2jYE4rOI3l9+AxO8c8VxcnEnSiNdLuaWLAtdsasw/FBjp2uM=
// yxon=1
// special=true
//

func (self *YoukuDownloader) getVideoInfo(vid String) error {
    var info JsonDict

    resp, err := self.session.Get("http://log.mmstat.com/yt.gif")
    if err != nil {
        return err
    }

    var cna String

LOOK_UP_CNA:
    for _, cs := range self.session.AllCookies() {
        for _, c := range cs {
            if c.Name == "cna" {
                cna = String(c.Value)

                nc := *c
                nc.Domain = ".youku.com"

                self.session.SetCookiesEx("http://youku.com", &nc)

                break LOOK_UP_CNA
            }
        }
    }

    if cna.IsEmpty() {
        self.Fatal("can't find cna")
    }

    resp, err = self.session.Get(
                    "http://ups.youku.com/ups/get.json",
                    http.Params(Dict{
                        "vid"       : vid,
                        "ccode"     : "0590",
                        "utid"      : cna,
                        "client_ip" : "192.168.1.1",
                        "client_ts" : time.Now().Unix(),
                        "ckey"      : "DIl58SLFxFNndSV1GFNnMQVYkx1PP5tKe1siZu/86PR1u/Wh1Ptd+WOZsHHWxysSfAOhNJpdVWsdVJNsfJ8Sxd8WKVvNfAS8aS8fAOzYARzPyPc3JvtnPHjTdKfESTdnuTW6ZPvk2pNDh4uFzotgdMEFkzQ5wZVXl2Pf1/Y6hLK0OnCNxBj3+nb0v72gZ6b0td+WOZsHHWxysSo/0y9D2K42SaB8Y/+aD2K42SaB8Y/+ahU+WOZsHcrxysooUeND",
                    }),
                )
    if err != nil {
        return err
    }

    err = resp.Json(&info)
    if err != nil {
        return err
    }

    self.parseVideoSegs(info)

    //self.videoInfo.security.sid, self.videoInfo.security.token = self.decryptSidAndToken(self.videoInfo.security.encryptString)
    fmt.Printf("%+v\n", spew.Sdump(self.videoInfo))

    return nil
}

func (self *YoukuDownloader) parseVideoSegs(info JsonDict) error {
    data := info.Map("data")

    self.title = data.Map("video").Str("title")

    //security := data.Map("security")
    //self.videoInfo.security.encryptString = security.Str("encrypt_string")
    //self.videoInfo.security.ip = int64(security.Int("ip"))

    stream := data.Array("stream")

    json.MustUnmarshal([]byte(`[{
                "audio_lang": "default",
                "milliseconds_audio": 2624442,
                "milliseconds_video": 2624664,
                "segs": [{
                    "size": 29796629,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/697411B0644377161DFE56A12/03000A070059FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D.mp4?ccode=0502&duration=390&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A106cb457d5eb2934ad8c0daedd4a5071&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 390400,
                    "total_milliseconds_audio": 390327,
                    "secret": "697411B0644377161DFE56A12",
                    "key": "5c6f4874176cea7b282ddf64",
                    "fileid": "03000A070059FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D"
                }, {
                    "size": 25322944,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/697493E6722377161DFE52AE1/03000A070159FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D.mp4?ccode=0502&duration=390&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A3c3bcaf038901284f72df8cc663837cb&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 390666,
                    "total_milliseconds_audio": 390652,
                    "secret": "697493E6722377161DFE52AE1",
                    "key": "6ef4a88fd2cdaffc2413a8e2",
                    "fileid": "03000A070159FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D"
                }, {
                    "size": 17460748,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6975161C7044A71DE62D22FC1/03000A070259FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D.mp4?ccode=0502&duration=389&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Aa890821cacf29d09c7187cb93973dfa5&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 389733,
                    "total_milliseconds_audio": 389746,
                    "secret": "6975161C7044A71DE62D22FC1",
                    "key": "39cc38e24197d0862413a8e2",
                    "fileid": "03000A070259FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D"
                }, {
                    "size": 11716502,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/65738F7AEDE39716EFB2739AE/03000A070359FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D.mp4?ccode=0502&duration=389&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A3e6166a48ab13d940345467401456f15&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 389200,
                    "total_milliseconds_audio": 389189,
                    "secret": "65738F7AEDE39716EFB2739AE",
                    "key": "296264dc316eb8ee2413a8e2",
                    "fileid": "03000A070359FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D"
                }, {
                    "size": 23904733,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/69761A8854B377161DFE52206/03000A070459FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D.mp4?ccode=0502&duration=395&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Af08cea79892903f7220c285fc972d9b9&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 395666,
                    "total_milliseconds_audio": 395668,
                    "secret": "69761A8854B377161DFE52206",
                    "key": "365477f366dbc8912413a8e2",
                    "fileid": "03000A070459FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D"
                }, {
                    "size": 21529381,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/657493E6E474B71E4F0733EEE/03000A070559FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D.mp4?ccode=0502&duration=387&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Ae84df92f9066d5c5ecb24f1dd0898186&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 387333,
                    "total_milliseconds_audio": 387332,
                    "secret": "657493E6E474B71E4F0733EEE",
                    "key": "813c9303087d91fc2620c423",
                    "fileid": "03000A070559FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D"
                }, {
                    "size": 19783244,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/69771EF4712307134008056F4/03000A070659FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D.mp4?ccode=0502&duration=281&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A15fbc68865a0001c6836420a6943f66f&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 281666,
                    "total_milliseconds_audio": 281528,
                    "secret": "69771EF4712307134008056F4",
                    "key": "4ab7ece72355f3d92413a8e2",
                    "fileid": "03000A070659FEEFF1C31401F9834DF308F658-3840-0335-5AE9-B5FF8F634E8D"
                }],
                "stream_ext": {
                    "hls_subtitle": "default",
                    "subtitle_lang": "default",
                    "one_seg_flag": 0,
                    "hls_logo": "none"
                },
                "size": 149514181,
                "subtitle_lang": "default",
                "media_type": "standard",
                "drm_type": "default",
                "stream_type": "mp4sd",
                "width": 480,
                "logo": "none",
                "m3u8_url": "http://pl-ali.youku.com/playlist/m3u8?vid=XMTEyMDM0OTQ4&type=flvhdv3&ups_client_netip=0e925fe2&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&ccode=0502&psid=0d4040fdf806bb07bdf7fefc6ef69488&duration=2624&expire=18000&drm_type=1&drm_device=7&ups_ts=1546101709&onOff=0&encr=0&ups_key=1df2f027e1e725325b6d6ac752487d0d",
                "height": 360
            }, {
                "audio_lang": "default",
                "milliseconds_audio": 2624440,
                "milliseconds_video": 2624557,
                "segs": [{
                    "size": 145515955,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6572E87849E34714E37035149/03000B07005AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD.mp4?ccode=0502&duration=394&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A25e1de599c7ccdea75d768ec2e9d05f0&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 394720,
                    "total_milliseconds_audio": 394692,
                    "secret": "6572E87849E34714E37035149",
                    "key": "e9c82ee7409fa391282ddf64",
                    "fileid": "03000B07005AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD"
                }, {
                    "size": 116609261,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/677516D287D4771CAB9F0231A/03000B07015AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD.mp4?ccode=0502&duration=386&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Ae26613bfc423bd7ebd7e7ab98c779e35&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 386360,
                    "total_milliseconds_audio": 386356,
                    "secret": "677516D287D4771CAB9F0231A",
                    "key": "115286c1d8f11974282ddf64",
                    "fileid": "03000B07015AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD"
                }, {
                    "size": 82328971,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6775D0F0B3F357154C4A44182/03000B07025AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD.mp4?ccode=0502&duration=392&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Ad3fa1b95b531115a6a450406141d1900&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 392479,
                    "total_milliseconds_audio": 392486,
                    "secret": "6775D0F0B3F357154C4A44182",
                    "key": "dee64219883c6c522620c423",
                    "fileid": "03000B07025AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD"
                }, {
                    "size": 63395517,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6977FF4A9653E718FBF4A2052/03000B07035AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD.mp4?ccode=0502&duration=392&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A2aae5aedd61208fff1620bc504c34002&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 392039,
                    "total_milliseconds_audio": 392045,
                    "secret": "6977FF4A9653E718FBF4A2052",
                    "key": "bb52671f6335b724282ddf64",
                    "fileid": "03000B07035AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD"
                }, {
                    "size": 112103579,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6777452C6C33871686D86379D/03000B07045AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD.mp4?ccode=0502&duration=390&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A8f08f6920f24aa835a6b5afb3d75296b&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 390000,
                    "total_milliseconds_audio": 390002,
                    "secret": "6777452C6C33871686D86379D",
                    "key": "799c96080f8e4ec82620c423",
                    "fileid": "03000B07045AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD"
                }, {
                    "size": 113465439,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6777FF4AA173871686D863C47/03000B07055AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD.mp4?ccode=0502&duration=387&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Ae6dd5b731dcb65d3059f95abf34c6b08&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 387280,
                    "total_milliseconds_audio": 387262,
                    "secret": "6777FF4AA173871686D863C47",
                    "key": "8f4a9ee3da2568b2282ddf64",
                    "fileid": "03000B07055AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD"
                }, {
                    "size": 90687741,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6778B9684763B717C166824D6/03000B07065AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD.mp4?ccode=0502&duration=281&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A1469511e14c48d214fd70e07233adfe3&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 281679,
                    "total_milliseconds_audio": 281597,
                    "secret": "6778B9684763B717C166824D6",
                    "key": "7ac290412b47f237282ddf64",
                    "fileid": "03000B07065AE1A8ABC31401F9834D30E0B22F-7E71-67C6-A2F0-E36AAB179CCD"
                }],
                "stream_ext": {
                    "hls_subtitle": "default",
                    "subtitle_lang": "default",
                    "one_seg_flag": 0,
                    "hls_logo": "none"
                },
                "size": 724106463,
                "subtitle_lang": "default",
                "media_type": "standard",
                "drm_type": "default",
                "stream_type": "mp4hd2v2",
                "width": 1120,
                "logo": "none",
                "m3u8_url": "http://pl-ali.youku.com/playlist/m3u8?vid=XMTEyMDM0OTQ4&type=mp4hd2v3&ups_client_netip=0e925fe2&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&ccode=0502&psid=0d4040fdf806bb07bdf7fefc6ef69488&duration=2624&expire=18000&drm_type=1&drm_device=7&ups_ts=1546101709&onOff=0&encr=0&ups_key=1df2f027e1e725325b6d6ac752487d0d",
                "height": 904
            }, {
                "audio_lang": "default",
                "milliseconds_audio": 2624458,
                "milliseconds_video": 2624400,
                "segs": [{
                    "size": 19780805,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/697864607CC3C7182A4096A3E/030002070050E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.flv?ccode=0502&duration=413&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A37c16311862b38920f8f376605ec347a&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 413733,
                    "total_milliseconds_audio": 413733,
                    "secret": "697864607CC3C7182A4096A3E",
                    "key": "2648408fea60ad0b2620c423",
                    "fileid": "030002070050E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }, {
                    "size": 15275412,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/697970EC81F3271411BC247E3/030002070150E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.flv?ccode=0502&duration=416&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Aeb297b7490c221f204fa5ed7d3404bd4&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 416934,
                    "total_milliseconds_audio": 416937,
                    "secret": "697970EC81F3271411BC247E3",
                    "key": "b83adf905d25fd45282ddf64",
                    "fileid": "030002070150E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }, {
                    "size": 9954949,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/677864605A7357154C4A42876/030002070250E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.flv?ccode=0502&duration=362&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Af9c5cb3105c8ba51e1f45494feff3363&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 362466,
                    "total_milliseconds_audio": 362464,
                    "secret": "677864605A7357154C4A42876",
                    "key": "41051e1f0f48d04a282ddf64",
                    "fileid": "030002070250E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }, {
                    "size": 9981529,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/697B8A044703A717588C7225B/030002070350E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.flv?ccode=0502&duration=396&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A084730e7f4593b3273533a9ca7bfe323&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 396600,
                    "total_milliseconds_audio": 396596,
                    "secret": "697B8A044703A717588C7225B",
                    "key": "a69d1ed97654e7492413a8e2",
                    "fileid": "030002070350E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }, {
                    "size": 16126058,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/697C96907A04D71F20BB4688B/030002070450E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.flv?ccode=0502&duration=366&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Ac34a4586ea56f5ad383943c8f1ce3896&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 366734,
                    "total_milliseconds_audio": 366736,
                    "secret": "697C96907A04D71F20BB4688B",
                    "key": "d6966dd19474f2632413a8e2",
                    "fileid": "030002070450E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }, {
                    "size": 10579780,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/697DA31C6B7357154C4A4684D/030002070550E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.flv?ccode=0502&duration=345&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A21081c6730361bd155a0560a61d1c2e8&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 345466,
                    "total_milliseconds_audio": 345467,
                    "secret": "697DA31C6B7357154C4A4684D",
                    "key": "0116eb07a2e91d5f2413a8e2",
                    "fileid": "030002070550E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }, {
                    "size": 12558759,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/677C9690F763C7182A40930B9/030002070650E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.flv?ccode=0502&duration=322&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A861509c52c15e7497c3a4237e6123049&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 322467,
                    "total_milliseconds_audio": 322525,
                    "secret": "677C9690F763C7182A40930B9",
                    "key": "f153b7fbf4f16a7d2413a8e2",
                    "fileid": "030002070650E37B63C31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }],
                "stream_ext": {
                    "hls_subtitle": "default",
                    "subtitle_lang": "default",
                    "one_seg_flag": 0,
                    "hls_logo": "none"
                },
                "size": 94257292,
                "subtitle_lang": "default",
                "media_type": "standard",
                "drm_type": "default",
                "stream_type": "flvhd",
                "width": 448,
                "logo": "none",
                "m3u8_url": "http://pl-ali.youku.com/playlist/m3u8?vid=XMTEyMDM0OTQ4&type=flvhdv3&ups_client_netip=0e925fe2&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&ccode=0502&psid=0d4040fdf806bb07bdf7fefc6ef69488&duration=2624&expire=18000&drm_type=1&drm_device=7&ups_ts=1546101709&onOff=0&encr=0&ups_key=1df2f027e1e725325b6d6ac752487d0d",
                "height": 336
            }, {
                "audio_lang": "default",
                "milliseconds_audio": 2624577,
                "milliseconds_video": 2624664,
                "segs": [{
                    "size": 16951308,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6772C09AD1F4271A9F5CD3062/0300200700593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.mp4?ccode=0502&duration=392&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A4573aa072a4413455bf38227e7be20e4&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 392866,
                    "total_milliseconds_audio": 393020,
                    "secret": "6772C09AD1F4271A9F5CD3062",
                    "key": "7e49bcb10e3a3d352413a8e2",
                    "fileid": "0300200700593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }, {
                    "size": 13745214,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/65724B2B7F3337147A96235F5/0300200701593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.mp4?ccode=0502&duration=387&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A930ba0e511ff9b4d67ed05354ab1cc8b&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 387133,
                    "total_milliseconds_audio": 387123,
                    "secret": "65724B2B7F3337147A96235F5",
                    "key": "a6b5937faf64255c2620c423",
                    "fileid": "0300200701593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }, {
                    "size": 10038991,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6773AB786213271411BC24C9B/0300200702593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.mp4?ccode=0502&duration=391&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A0e592857fe3f0bd26f4ee572db03622c&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 391133,
                    "total_milliseconds_audio": 391116,
                    "secret": "6773AB786213271411BC24C9B",
                    "key": "5933a198bc55b6b4282ddf64",
                    "fileid": "0300200702593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }, {
                    "size": 6624783,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/69750BC5BB64671C42C503FA0/0300200703593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.mp4?ccode=0502&duration=392&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Adb99c8562df45a201e6510d2aad6d3a2&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 392333,
                    "total_milliseconds_audio": 392324,
                    "secret": "69750BC5BB64671C42C503FA0",
                    "key": "8855b0a9aba6017e282ddf64",
                    "fileid": "0300200703593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }, {
                    "size": 14582180,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6573AB78FF934714E37032633/0300200704593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.mp4?ccode=0502&duration=391&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Ab909a6f7b4af409c6554e843fecaa8c3&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 391533,
                    "total_milliseconds_audio": 391534,
                    "secret": "6573AB78FF934714E37032633",
                    "key": "8c57394890e32ffd282ddf64",
                    "fileid": "0300200704593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }, {
                    "size": 10433790,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/657420E7F6336715B524434C2/0300200705593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.mp4?ccode=0502&duration=388&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A9fabb790ea2601e962489ae69342d95c&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 388000,
                    "total_milliseconds_audio": 388005,
                    "secret": "657420E7F6336715B524434C2",
                    "key": "9055853bc86932a3282ddf64",
                    "fileid": "0300200705593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }, {
                    "size": 10437749,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/65749656B21337147A962635D/0300200706593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE.mp4?ccode=0502&duration=281&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A5511ac60d005e2d6f8c55930e668be85&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 281666,
                    "total_milliseconds_audio": 281455,
                    "secret": "65749656B21337147A962635D",
                    "key": "dcba126e8b455126282ddf64",
                    "fileid": "0300200706593088BCC31401F9834D4C741DF1-BA40-16E5-3F74-74BE155A4FEE"
                }],
                "stream_ext": {
                    "hls_subtitle": "default",
                    "subtitle_lang": "default",
                    "one_seg_flag": 0,
                    "hls_logo": "none"
                },
                "size": 82814015,
                "subtitle_lang": "default",
                "media_type": "standard",
                "drm_type": "default",
                "stream_type": "3gphd",
                "width": 332,
                "logo": "none",
                "m3u8_url": "http://pl-ali.youku.com/playlist/m3u8?vid=XMTEyMDM0OTQ4&type=3gphdv3&ups_client_netip=0e925fe2&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&ccode=0502&psid=0d4040fdf806bb07bdf7fefc6ef69488&duration=2624&expire=18000&drm_type=1&drm_device=7&ups_ts=1546101709&onOff=0&encr=0&ups_key=1df2f027e1e725325b6d6ac752487d0d",
                "height": 268
            }, {
                "audio_lang": "default",
                "milliseconds_audio": 2624441,
                "milliseconds_video": 2624558,
                "segs": [{
                    "size": 71827187,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6972FD587294A71DE62D23E4E/030008070059FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10.mp4?ccode=0502&duration=394&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Aae1f3175f6dcc40e1aad6151161374cd&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 394680,
                    "total_milliseconds_audio": 394669,
                    "secret": "6972FD587294A71DE62D23E4E",
                    "key": "57176b96e1c9152f2413a8e2",
                    "fileid": "030008070059FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10"
                }, {
                    "size": 57584000,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/67729DAD8FB3C7182A4095FEF/030008070159FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10.mp4?ccode=0502&duration=386&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A2a334045efb70fec37d0723192c26f2a&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 386440,
                    "total_milliseconds_audio": 386426,
                    "secret": "67729DAD8FB3C7182A4095FEF",
                    "key": "0f6edeaa48d51af82413a8e2",
                    "fileid": "030008070159FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10"
                }, {
                    "size": 38073769,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6973BCAE5194571BD9EAF51DD/030008070259FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10.mp4?ccode=0502&duration=389&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A8826063c54ea41e1dcea4249972503d7&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 389560,
                    "total_milliseconds_audio": 389561,
                    "secret": "6973BCAE5194571BD9EAF51DD",
                    "key": "e2bede5e4a917fd9282ddf64",
                    "fileid": "030008070259FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10"
                }, {
                    "size": 28204770,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/67735D03B443C7182A409474F/030008070359FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10.mp4?ccode=0502&duration=394&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A07c83368c4816511b0b65405152a244c&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 394919,
                    "total_milliseconds_audio": 394924,
                    "secret": "67735D03B443C7182A409474F",
                    "key": "b6ff3019e1e4606e282ddf64",
                    "fileid": "030008070359FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10"
                }, {
                    "size": 52729842,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/69747C044344671C42C50312C/030008070459FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10.mp4?ccode=0502&duration=390&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=A7c8c0db7cc49ec3848eb97750300eef7&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 390000,
                    "total_milliseconds_audio": 390002,
                    "secret": "69747C044344671C42C50312C",
                    "key": "b8cbb8372438aa4f282ddf64",
                    "fileid": "030008070459FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10"
                }, {
                    "size": 53487665,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/6974DBAF57A4E71F8995538AE/030008070559FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10.mp4?ccode=0502&duration=387&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Aabf1f9f9b616382d39b8b6f9577724f3&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 387280,
                    "total_milliseconds_audio": 387262,
                    "secret": "6974DBAF57A4E71F8995538AE",
                    "key": "6123faedfbed1f992620c423",
                    "fileid": "030008070559FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10"
                }, {
                    "size": 45986601,
                    "cdn_url": "http://vali-dns.cp31.ott.cibntv.net/69753B5A5DA3B717C16683F6B/030008070659FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10.mp4?ccode=0502&duration=281&expire=18000&psid=0d4040fdf806bb07bdf7fefc6ef69488&ups_client_netip=0e925fe2&ups_ts=1546101709&ups_userid=&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&vid=XMTEyMDM0OTQ4&vkey=Ab9b845870467d3271bd0053b8e9f1656&s=cbffd26c962411de83b1&sp=",
                    "total_milliseconds_video": 281679,
                    "total_milliseconds_audio": 281597,
                    "secret": "69753B5A5DA3B717C16683F6B",
                    "key": "b4dc3d068a052735282ddf64",
                    "fileid": "030008070659FEEFF3C31401F9834D66F9AFBB-AE18-1070-385E-26E08A64CA10"
                }],
                "stream_ext": {
                    "hls_subtitle": "default",
                    "subtitle_lang": "default",
                    "one_seg_flag": 0,
                    "hls_logo": "none"
                },
                "size": 347893834,
                "subtitle_lang": "default",
                "media_type": "standard",
                "drm_type": "default",
                "stream_type": "mp4hd",
                "width": 720,
                "logo": "none",
                "m3u8_url": "http://pl-ali.youku.com/playlist/m3u8?vid=XMTEyMDM0OTQ4&type=mp4hdv3&ups_client_netip=0e925fe2&utid=y5GuFNGashgCAQ6SX%2BJhN8kS&ccode=0502&psid=0d4040fdf806bb07bdf7fefc6ef69488&duration=2624&expire=18000&drm_type=1&drm_device=7&ups_ts=1546101709&onOff=0&encr=0&ups_key=1df2f027e1e725325b6d6ac752487d0d",
                "height": 540
            }]`), &stream)

    type StreamType struct {
        name        String
        profile     String
        container   String
        priority    int
        hd          int
    }

    getStreamType := func (stream JsonDict) StreamType {
        var st StreamType

        switch t := stream.Str("stream_type"); t {
            case "mp4hd3", "hd3":
                st.priority = -6
                st.profile = "1080P"
                st.container = "flv"
                st.hd = 3

            case "mp4hd2", "hd2", "mp4hd2v2":
                st.priority = 5
                st.profile = "超清"
                st.container = "flv"
                st.hd = 2

            case "mp4hd", "mp4":
                st.priority = 4
                st.profile = "高清"
                st.container = "mp4"
                st.hd = 1

            case "mp4sd":
                st.priority = 2
                st.profile = "标清"
                st.container = "mp4"
                st.hd = 0

            case "flvhd", "flv":
                st.priority = 2
                st.profile = "标清"
                st.container = "flv"
                st.hd = 0

            case "3gphd":
                st.priority = 1
                st.profile = "标清（3GP）"
                st.container = "3gp"
                st.hd = 0

            default:
                self.Critical("unsupported stream type: %v", t)
                panic(nil)
        }

        return st
    }

    var streamType StreamType

    for index := range stream {
        s := stream.Map(index)

        st := getStreamType(s)
        if st.priority < streamType.priority {
            continue
        }

        streamType = st

        segs := s.Array("segs")

        fmt.Printf("stream_type: %+v\n", streamType)

        self.videoInfo.segs = nil

        for index := range segs {
            seg := segs.Map(index)

            self.videoInfo.segs = append(self.videoInfo.segs, YoukuVideoInfoSeg{
                totalMillisecondsAudio  : seg.Int64("total_milliseconds_audio"),
                totalMillisecondsVideo  : seg.Int64("total_milliseconds_video"),
                size                    : seg.Int64("size"),
                cdn_url                 : seg.Str("cdn_url"),
                fileid                  : seg.Str("fileid"),
                key                     : seg.Str("key"),
                hd                      : streamType.hd,
                container               : streamType.container,
            })
        }
    }

    return nil
}

func (self *YoukuDownloader) parseVideoInfo() error {
    for index, seg := range self.videoInfo.segs {
        // com\youku\utils\GetUrl.as

        //url := fmt.Sprintf("http://k.youku.com/player/getFlvPath/sid/%s_00/st/%s/fileid/%s", self.videoInfo.security.sid, seg.container, seg.fileid)
        //resp, err := self.session.Get(
        //                url,
        //                http.Params(Dict{
        //                    "start"     : "0",
        //                    "K"         : seg.key,
        //                    "hd"        : seg.hd,
        //                    "myp"       : "0",
        //                    "ts"        : "64",
        //                    "ypp"       : "0",      // P2PConfig.ypp @ com\youku\P2PConfig.as
        //                    "ctype"     : "10",     // PlayerConstant.CTYPE @ com\youku\data\PlayerConstant.as
        //                    "ev"        : "1",      // PlayerConstant.EV @ com\youku\data\PlayerConstant.as
        //                    "token"     : self.videoInfo.security.token,
        //                    "oip"       : self.videoInfo.security.ip,
        //                    "ep"        : self.encryptEp(self.videoInfo, seg.fileid),
        //                    "yxon"      : "1",
        //                    "special"   : "true",
        //                }),
        //                http.Headers(Dict{
        //                    "X-Requested-With"  : "ShockwaveFlash/25.0.0.171",
        //                    "Accept"            : "*/*",
        //                    "Referer"           : self.url,
        //                }),
        //                http.Ignore404(false),
        //            )
        //if err != nil {
        //    self.Critical("get seg %d failed: %v", index, err)
        //    panic(nil)
        //    return err
        //}

        //var r JsonArray
        //
        //err = resp.Json(&r)
        //if err != nil {
        //    self.Critical("get seg %d failed: %v", index, err)
        //    panic(nil)
        //    return err
        //}

        if len(self.videoInfo.segs) == 1 {
            self.links = append(self.links, DownloadLink{
                url : seg.cdn_url,
                name: String(fmt.Sprintf("%s.%s", self.title, seg.container)),
            })
        } else {
            self.links = append(self.links, DownloadLink{
                url : seg.cdn_url,
                name: String(fmt.Sprintf("%s.part%02d.%s", self.title, index + 1, seg.container)),
            })
        }
    }

    return nil
}

func (self *YoukuDownloader) decryptSidAndToken(encryptString String) (sid, token String) {
    data := base64.DecodeString(encryptString.String())

    cipher, _ := des.NewCipher([]byte("00149ad5"))
    for i := 0; i < len(data); i += cipher.BlockSize() {
        cipher.Decrypt(data[i:], data[i:])
    }

    decrypted := String(data).Split("\x00", 1)[0].Split("_", 1)

    sid = decrypted[0]
    token = decrypted[1]
    return
}

//func (self *YoukuDownloader) encryptEp(info YoukuVideoInfo, fileId String) String {
//    bctime := 0
//    ep := fmt.Sprintf("%v_%v_%v_%v", info.security.sid, fileId, info.security.token, bctime)
//    sum := md5.Sum([]byte(ep + "_kservice"))
//
//    ep = ep + "_" + fmt.Sprintf("%x", sum[:])[:4]
//
//    data := []byte(ep)
//
//    cipher, _ := des.NewCipher([]byte("21dd8110"))
//
//    for len(data) % cipher.BlockSize() != 0 {
//        data = append(data, 0)
//    }
//
//    for i := 0; i < len(data); i += cipher.BlockSize() {
//        cipher.Encrypt(data[i:], data[i:])
//    }
//
//    return base64.EncodeToString(data)
//}
