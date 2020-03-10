package main

import (
    "encoding/csv"
    "net/http"
    "io"
    "fmt"
    "time"
    "sync"
    "net/url"
    "math"
    "io/ioutil"
)

const (
    iexBaseUri = "https://cloud.iexapis.com/stable/"
    iexSymbolsUri = iexBaseUri + "/ref-data/iex/symbols/"
)

type Symbol string

type Ratelimit struct {
    LastRequest time.Time
    Interval time.Duration
    sync.RWMutex
}

func (R Ratelimit) Sig() {
    R.Lock()
    defer R.Unlock()
    R.LastRequest = time.Now()
}

func (R Ratelimit) Backoff() {
    R.RLock()
    defer R.RUnlock()
    sleep := time.Since(R.LastRequest) - R.Interval
    time.Sleep(sleep)
}

func (R Ratelimit) Ready() {
    R.Backoff()
    R.Sig()
}

type IEX struct {
    Token string
    Ratelimit
}

func NewIEX(secret string) IEX {
    return IEX{
        Token: secret,
        Ratelimit: Ratelimit{Interval: 10 * time.Millisecond},
    }
}

func (api *IEX) Tickers() (T []Symbol, err error) {
    api.Ready()
    T = make([]Symbol, 0, 8192)
    uri := fmt.Sprintf("%s?format=csv&token=%s",
        iexSymbolsUri, api.Token)
    rsp, err := http.Get(uri)
    if err != nil {
        return
    }
    r := csv.NewReader(rsp.Body)
    var row []string
    for {
        row, err = r.Read()
        if err != nil {
            if err == io.EOF {
                err = nil
            }
            return
        }
        if row[2] == "true" {
            T = append(T, Symbol(row[0]))
        }
    }
    return
}

func (api *IEX) Intraday(sym string, resolution time.Duration, date time.Time) (r io.Reader, err error) {
    n := float64(resolution) / float64(time.Minute)
    n = math.Ceil(n)
    // resolution = time.Hour <=> n = 60
    if n <= 1 {
        n = 1
    }
    k := int(n)
    exactDate := fmt.Sprintf("%d%d%d",
        date.Year(), date.Month(), date.Day())
    params := map[string]interface{} {
        "chartIEXOnly": true,
        "chartInterval": k,
        "exactDate": exactDate,
        "includeToday": true,
        "token": api.Token,
    }
    uri := fmt.Sprintf("%s/stock/%s/intraday-prices?",
        iexBaseUri, sym)
    for k, v := range params {
        uri += url.QueryEscape(fmt.Sprintf("%s=%v&", k, v))
    }
    uri = uri[:len(uri)-1]
    rsp, err := http.Get(uri)
    if rsp.StatusCode != 200 && err == nil {
        var buf []byte
        buf, err = ioutil.ReadAll(rsp.Body)
        if err != nil {
            err = fmt.Errorf("read error payload: %s", err)
            return
        }
        err = fmt.Errorf("IEX Cloud returned HTTP/%s: %s", rsp.Status, string(buf))
        return
    }
    r = rsp.Body
    return
}
