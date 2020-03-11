package iex

import (
	"crypto/tls"
	"encoding/csv"
	"fmt"
	gj "github.com/tidwall/gjson"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Symbol string

type Candle struct {
	time.Time
	High, Low              float64
	Open, Close            float64
	Average                float64
	Notional               float64
	Volume, NumberOfTrades int
}

func (C *Candle) UnmarshalJSON(buf []byte) (err error) {
	keys := []string{
		".minute", ".high", ".low", ".open", ".close", ".average", ".notional",
		".volume", ".numberOfTrades",
	}
	results := gj.GetManyBytes(buf, keys...)
	floats := []*float64{
		&C.High, &C.Low, &C.Open, &C.Close, &C.Average, &C.Notional,
	}
	ints := []*int{&C.Volume, &C.NumberOfTrades}
	for i := range floats {
		*floats[i] = results[i].Float()
	}
	for i := range ints {
		*ints[i] = int(results[i+len(floats)].Int())
	}
	return
}

func (C *Candle) Tabulate(steno *csv.Writer) {
	fields := []interface{}{
		C.Minute, C.High, C.Low, C.Open, C.Close, C.Average, C.Volume,
		C.Notional, C.NumberOfTrades,
	}
	row := make([]string, len(fields))
	for i, x := range fields {
		row[i] = fmt.Sprintf("%v", x)
	}
	steno.Write(row)
}

type Ratelimit struct {
	C <-chan time.Time
}

func NewRatelimit(interval time.Duration) Ratelimit {
	return Ratelimit{time.NewTicker(interval).C}
}

func (R Ratelimit) Ready() {
	<-R.C
}

type IEX struct {
	Token string
	Ratelimit
	*http.Client
}

func New(secret string) *IEX {
	return &IEX{
		Token:     secret,
		Ratelimit: NewRatelimit(10 * time.Millisecond),
		Client: &http.Client{
			Transport: &http.Transport{
				TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
			},
		},
	}
}

func (api *IEX) Tickers() (T []Symbol, err error) {
	api.Ready()
	T = make([]Symbol, 0, 8192)
	uri := fmt.Sprintf("%s?format=csv&token=%s",
		symbUri, url.QueryEscape(api.Token))
	rsp, err := http.Get(uri)
	if err != nil {
		return
	}
	r := csv.NewReader(rsp.Body)
	r.ReuseRecord = true
	r.Read()
	var row []string
	for {
		row, err = r.Read()
		if err != nil {
			if err == io.EOF && row == nil {
				err = nil
			} else {
				err = fmt.Errorf("parse csv: %s", err)
			}
			return
		}
		if row[2] == "true" {
			T = append(T, Symbol(row[0]))
		}
	}
	return
}
