package main

import (
	"github.com/jessevdk/go-flags"
)

type Opts struct {
	Launch  bool   `long:"launch"            description:"Create environment"`
	System  string `long:"system"  short:"s" description:"Override system value"`
	Verbose bool   `long:"verbose" short:"v" description:"Increase logging verbosity"`
	Version bool   `long:"version"           description:"Show version"`
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
