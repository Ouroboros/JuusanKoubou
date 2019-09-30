package main

import (
    . "ml/dict"
    . "ml/strings"

    "fmt"
    "sync"
    "time"
    "math/rand"

    "github.com/PuerkitoBio/goquery"

    "ml/trace"
    "ml/net/http"
    "ml/logging/logger"
)

const (
    BASE_URL    = "https://bbs.saraba1st.com/2b"
    FORUM_URL   = "https://bbs.saraba1st.com/2b/forum-75-1.html"
)

type Account struct {
    UserName    String              `json:"username,omitempty"`
    Password    String              `json:"password,omitempty"`
    Cookie      String              `json:"cookie,omitempty"`
}

type Saraba1stClient struct {
    *http.Session
    logger.BaseLogger
    account *Account
}

func NewSaraba1stClient(acc *Account) *Saraba1stClient {
    sess := http.NewSession(
        http.EnableHTTP2(),
        http.DefaultHeaders(Dict{
            http.HeaderKey_UserAgent        : "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3865.90 Safari/537.36",
            http.HeaderKey_Accept           : "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3",
            http.HeaderKey_AcceptEncoding   : "gzip, deflate, br",
            http.HeaderKey_AcceptLanguage   : "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7,zh-TW;q=0.6",
        }),
    )

    //sess.SetHTTPProxyString("http://127.0.0.1:6789")

    c := &Saraba1stClient{
        Session : sess,
        account : acc,
    }

    c.BaseLogger.Prefix = fmt.Sprintf("[%s] ", acc.UserName)

    return c
}

func (self *Saraba1stClient) Login() error {
    resp, err := self.Post(
                    fmt.Sprintf("%s/member.php", BASE_URL),
                    http.Headers(Dict{
                       http.HeaderKey_ContentType : http.HeaderValue_FormURLEncoded,
                    }),
                    http.Params(Dict{
                        "mod"           : "logging",
                        "action"        : "login",
                        "loginsubmit"   : "yes",
                        "infloat"       : "yes",
                        "lssubmit"      : "yes",
                        "inajax"        : "1",
                    }),
                    http.Body(Dict{
                        "fastloginfield"    : "username",
                        "username"          : self.account.UserName,
                        "cookietime"        : "2592000",
                        "password"          : self.account.Password,
                        "quickforward"      : "yes",
                        "handlekey"         : "ls",
                    }),
                )

    if err != nil {
        return err
    }

    if resp.StatusCode != http.StatusOK {
        return trace.NewBaseException("resp.Status: %v", resp.Status)
    }

    if String(resp.Content).Find(`<script type="text/javascript" reload="1">window.location.href='`) == -1 {
        return trace.NewBaseException("resp.Content: %s", resp.Content)
    }

    return nil
}

func (self *Saraba1stClient) loadDoc(url string) (*goquery.Document, error) {
    resp, err := self.Get(url)
    if err != nil {
        return nil, err
    }

    doc, err := goquery.NewDocumentFromReader(resp.Text().NewReader())
    if err != nil {
        return nil, err
    }

    return doc, nil
}

func (self *Saraba1stClient) OpenThread() error {
    doc, err := self.loadDoc(FORUM_URL)
    if err != nil {
        return err
    }

    var threads []*goquery.Selection

    doc.Find("tbody").Each(func(i int, s *goquery.Selection) {
        id_, exist := s.Attr("id")
        id := String(id_)
        if exist && id.StartsWith("normalthread_") {
            threads = append(threads, s)
        }
    })

    if len(threads) == 0 {
        self.Critical("empty thread")
        return nil
    }

    t := threads[rand.Intn(len(threads))].Find("a.s.xst")
    href, ok := t.Attr("href")

    if ok == false {
        self.Critical("can't find thread addr")
        return nil
    }

    credit := String(doc.Find("#extcreditmenu").Text())
    usergroup := String(doc.Find("#g_upmine").Text())

    self.Info("%s @ %-12s %s", credit, usergroup, t.Text())

    resp, err := self.Get(
                    fmt.Sprintf("%s/%s", BASE_URL, href),
                    http.IgnoreResponseBody(true),
                )

    if err != nil {
        return err
    }

    resp.Close()

    study_daily := doc.Find("#loginstatus")
    study_daily = study_daily.SiblingsFiltered("a[href*='study_daily']")
    if len(study_daily.Nodes) == 0 {
        return nil
    }

    href, ok = study_daily.Attr("href")
    if ok == false {
        self.Debug("study_daily not found: %v", study_daily.Text())
        return nil
    }

    study_daily_url := fmt.Sprintf("%s/%s", BASE_URL, href)
    self.Notice("study_daily_url: %s", study_daily_url)

    _, err = self.Get(study_daily_url)

    return err
}

func run(acc *Account) error {
    c := NewSaraba1stClient(acc)

    err := c.Login()
    if err != nil {
        c.Critical("login: %v", err)
        return err
    }

    for {
        err = c.OpenThread()
        if err != nil {
            c.Critical("OpenThread: %v", err)
            time.Sleep(time.Second * 10)
            continue
        }

        time.Sleep(time.Second * 600)
    }
}

func main() {
    accs := [...]Account{
    }

    wg := sync.WaitGroup{}
    wg.Add(len(accs))

    for i := range accs {
        go func(acc *Account) {
            defer wg.Done()
            run(acc)
        }(&accs[i])
    }

    wg.Wait()
}
