package main

import (
	"log"
	"strconv"
	"time"

	"github.com/jpillora/goff/ff"
	"github.com/jpillora/opts"
)

//built is the build time in unix-epoch seconds
var built = "00000000"

func main() {
	c := ff.Config{
		OutputFormat: "adts",
		OutputType:   "m4a",
		MaxBitrate:   48,
	}
	if n, _ := strconv.ParseInt(built, 10, 64); n > 0 {
		built = time.Unix(n, 0).String()
	}
	opts.New(&c).Version(built).Parse()
	if err := ff.Concat(c); err != nil {
		log.Fatal(err)
	}
}
