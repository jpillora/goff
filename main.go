package main

import (
	"log"

	"github.com/jpillora/goff/ff"
	"github.com/jpillora/opts"
)

var version = "0.0.0"

func main() {
	c := ff.Config{
		OutputFormat: "adts",
		OutputType:   "m4a",
		MaxBitrate:   64,
	}
	opts.New(&c).Version(version).Parse()
	if err := ff.Concat(c); err != nil {
		log.Fatal(err)
	}
}
