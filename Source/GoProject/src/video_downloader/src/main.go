package main

import (
    . "ml/strings"

    "fmt"
    urllib "net/url"

    "ml/os2"
    "ml/trace"
    "ml/console"

    "downloader"
    "path/filepath"
)

func getDownloaderFromUrl(url String) downloader.Downloader {
    if url.ToLower().StartsWith("http") == false {
        url = "http://" + url
    }

    u, err := urllib.Parse(url.String())
    if err != nil {
        trace.Raise(trace.NewWrapper(err, "parse url '%v' failed:", url))
    }

    fmt.Println(u.Host)

    switch String(u.Host).ToLower() {
        case "v.ku6.com", "baidu.ku6.com":
            return downloader.NewKu6(url)

        case "v.youku.com":
            return downloader.NewYouku(url)

        case "open.163.com":
            return downloader.NewNetEase(url)

        case "www.acfun.cn":
            return downloader.NewAcFun(url)

        default:
            trace.Raise(trace.NewNotImplementedError("%v unimplemented", url))
            return nil
    }
}

func run() {
    var url String

    fmt.Print("url = ")
    fmt.Scanf("%s\n", &url)

    url = "https://v.youku.com/v_show/id_XMTEyMDM0OTQ4.html"

    if url.IsEmpty() {
        url = "http://v.youku.com/v_show/id_XMTg3Nzk5MzAzMg==.html?qq-pf-to=pcqq.c2c"
    }

    d := getDownloaderFromUrl(url)
    defer d.Close()

    path := String(filepath.Join(os2.ExecutablePath(), "已下载"))

    result, err := d.Analysis()
    if err != nil {
        fmt.Println(err)
        return
    }

    switch result {
        case downloader.AnalysisSuccess:
            _, err = d.Download(path)
    }

    if err != nil {
        fmt.Println(err)
        fmt.Println("下载失败")
        return
    }
}

func main() {
    if e := trace.Try(run); e != nil {
        fmt.Println(e)
    }

    console.Pause("完成")
}
