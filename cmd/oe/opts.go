package main

import (
	"github.com/jessevdk/go-flags"
)

type Opts struct {
	Launch  bool   `long:"launch" description:"create environment"`
	Verbose bool   `long:"verbose" short:"v"`
	System  string `long:"system" short:"s"`
	Params  []string
}

func GetOpts(args []string) (Opts, error) {
	opts := Opts{}
	params, err := flags.ParseArgs(&opts, args)
	if err != nil {
		return Opts{}, err
	}
	opts.Params = params
	return opts, nil
}
