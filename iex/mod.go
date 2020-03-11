package iex

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseUri = "https://cloud.iexapis.com/stable/"
	symbUri = baseUri + "/ref-data/iex/symbols/"
)

type IEXError struct {
	*http.Response
	Op string
}

func (err IEXError) Error() string {
	return fmt.Sprintf("%s: IEX Cloud returned %s", err.Op, err.Status)
}

func iexError(op string, rsp *http.Response) (err *IEXError) {
	if rsp.StatusCode < 200 || rsp.StatusCode > 299 {
		err = &IEXError{rsp, op}
	}
	return
}

type IEXReq interface {
	Marshal() *http.Request
	Unmarshal(rsp *http.Response) (interface{}, error)
	Op() string
}

type QArgs map[string]interface{}

func (q QArgs) String() string {
	b := strings.Builder{}
	for k, v := range q {
		inline := url.QueryEscape(fmt.Sprintf("%v", v))
		b.WriteString(fmt.Sprintf("%s=%s&", k, inline))
	}
	s := b.String()
	return s[:len(s)-1]
}

func (api *IEX) Execute(req IEXReq) (obj interface{}, err error) {
	api.Ready()
	rsp, err := api.Do(req.Marshal())
	if err != nil {
		return
	}
	obj, err = req.Unmarshal(rsp)
	return
}

func (api *IEX) Intraday(sym Symbol, resolution time.Duration, date time.Time) (candles []Candle, err error) {
	api.Ready()
	req := Intraday{sym, newIntradayArgs(api.Token, resolution, date)}
	obj, err := api.Execute(&req)
	candles = obj.([]Candle)
	return
}

func (api *IEX) IntradayBatch(syms []Symbol, resolution time.Duration, date time.Time) (chandelier map[Symbol][]Candle, err error) {
	batch := IntradayBatch{syms, newIntradayArgs(api.Token, resolution, date)}
	obj, err := api.Execute(&batch)
	chandelier = obj.(map[Symbol][]Candle)
	return
}
