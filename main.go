package main

import (
    "os"
    "io"
    "flag"
    "fmt"
    "strings"
    "time"
)

var (
    assetsOnly bool
    daycount uint
    secret string
    hsyms string
    hres string
)

func parseResolution(hres string) (res time.Duration, err error) {
    units := strings.NewReader(hres)
    var (
        x float64
        u rune)
    for {
        _, err = fmt.Fscanf(units, "%f%c", &x, &u)
        if err != nil {
            if err == io.EOF {
                err = nil
                return
            }
            err = fmt.Errorf("Scan units: %s", err)
            return
        }
        switch u {
        case 'd':
            x *= 24
            fallthrough
        case 'h':
            x *= 60
            fallthrough
        case 'm':
            x *= 60
            fallthrough
        case 's':
            x *= float64(time.Second)
        default:
            err = fmt.Errorf("unit not recognized: '%c'", u)
            return
        }
        res += time.Duration(x)
    }
}

func main() {
    flag.BoolVar(&assetsOnly, "list", true,
                 "specify whether to supply only assets to stdout")
    flag.UintVar(&daycount, "days", 30,
                 "specify the number of days to retrieve and serialize")
    flag.StringVar(&secret, "secret", "",
                   "specify the secret for IEX Cloud API")
    flag.StringVar(&hres, "res", "1m",
        "resolution of intraday data, e.g.: 1.5h, 1h30m (default: 1m)")
    flag.StringVar(&hsyms, "syms", "",
                   "comma-delimited symbols; if not provided, print "+
                   "tradable symbols on stdout")

    flag.Parse()
    if secret == "" {
        secret = os.Getenv("IEX_CLOUD_SECRET")
    }
    if secret == "" {
        fmt.Fprintf(os.Stderr, "please provide an IEX cloud secret: "+
            "either a -secret flag on the command-line or set "+
            "environment variable IEX_CLOUD_SECRET=<YOURS>\n")
        os.Exit(1)
    }
    res, err := parseResolution(hres)
    if err != nil {
        err = fmt.Errorf("parse resolution specifier '%s': %s\n", hres, err)
        os.Exit(1)
    }
    c := NewIEX(secret)
    // generate candidate dates; create date/sym directory tree for each
    // date retrieved in working directory
}
