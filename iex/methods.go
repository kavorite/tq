package iex

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type IntradayArgs struct {
	Endpoint, Token string
	Date            time.Time
	QArgs
}

type Intraday struct {
	Symbol
	*IntradayArgs
}

type IntradayBatch struct {
	Symbols []Symbol
	*IntradayArgs
}

func newIntradayArgs(token string, resolution time.Duration, date time.Time) *IntradayArgs {
	n := float64(resolution) / float64(time.Minute)
	n = math.Ceil(n)
	// resolution = time.Hour <=> n = 60
	if n <= 1 {
		n = 1
	}
	date = date.Truncate(time.Hour * 24)
	return &IntradayArgs{
		Endpoint: "intraday-prices",
		Date:     date,
		Token:    token,
		QArgs: QArgs{
			"chartIEXOnly":  true,
			"includeToday":  true,
			"chartInterval": int(n),
			"exactDate":     date,
		},
	}
}

func (req *Intraday) Marshal() (r *http.Request) {
	req.QArgs["token"] = url.QueryEscape(req.Token)
	uri := fmt.Sprintf("%s/stock/%s/%s?%s", baseUri, req.Endpoint, req.Symbol, req.QArgs)
	r, _ = http.NewRequest("GET", uri, nil)
	return
}

func (req *Intraday) Op() string {
	return fmt.Sprintf("hydrate intraday data for IEX:%s", req.Symbol)
}

func (req *Intraday) Unmarshal(rsp *http.Response) (obj interface{}, err error) {
	err = iexError(req.Op(), rsp)
	if err != nil {
		return
	}
	buf, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return
	}
	candles := make([]Candle, 0, len(buf)/128)
	err = json.Unmarshal(buf, &candles)
	if err != nil {
		return
	}
	k := 0
	for _, c := range candles {
		if c.NumberOfTrades != 0 {
			candles[k] = c
			// annotate each candle's time with its date of record
			candles[k].Time = candles[k].Time.Add(req.Date.Truncate(time.Hour * 24).Sub(time.Time{}))
			k++
		}
	}
	candles = candles[:k]
	if len(candles) == 0 {
		return
	}
	obj = candles
	return
}

type Tickers struct {
	Token string
}

func (req Tickers) Marshal() (r *http.Request) {
	uri := fmt.Sprintf("%s?format=csv&token=%s", symbUri, url.QueryEscape(req.Token))
	r, _ = http.NewRequest("GET", uri, nil)
	return
}

func (req Tickers) Unmarshal(rsp *http.Response) (obj interface{}, err error) {
	T := make([]Symbol, 0, 1<<20)
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
	}
	if row[2] == "true" {
		T = append(T, Symbol(row[0]))
	}
	obj = T
	return
}

func (batch *IntradayBatch) Len() int {
	return len(batch.Symbols)
}

func (batch *IntradayBatch) Op() string {
	return "batch request intra-day quotes"
}

func (batch *IntradayBatch) Marshal() (r *http.Request) {
	if batch.Len() > 100 {
		panic("batch too large")
	}
	params := make(QArgs, 16)
	endpointSet := make(map[string]struct{}, 16)
	tickers := make([]string, batch.Len())
	for i, t := range batch.Symbols {
		tickers[i] = string(t)
	}
	endpoints := make([]string, 0, len(endpointSet))
	for endpoint := range endpointSet {
		endpoints = append(endpoints, endpoint)
	}
	params["types"] = strings.Join(endpoints, ",")
	params["symbols"] = strings.Join(tickers, ",")
	uri := fmt.Sprintf("%s/market/batch?%s", baseUri, params)
	r, _ = http.NewRequest("GET", uri, nil)
	return
}

func (batch *IntradayBatch) Unmarshal(rsp *http.Response) (obj interface{}, err error) {
	dict := make(map[Symbol][]Candle, 128*batch.Len())
	buf, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(buf, dict)
	if err == nil {
		obj = dict
	}
	return
}
