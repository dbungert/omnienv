package main

import (
	"github.com/dbungert/omnienv/internal/omnienv"
	"github.com/jessevdk/go-flags"
)

func GetOpts(args []string) (omnienv.Opts, error) {
	opts := omnienv.Opts{}

	parser := flags.NewParser(
		&opts,
		flags.HelpFlag|flags.PrintErrors|flags.PassDoubleDash|flags.PassAfterNonOption,
	)
	parser.Usage = "[OPTIONS]"

	params, err := parser.ParseArgs(args)
	if err != nil {
		return omnienv.Opts{}, err
	}
	opts.Params = params
	return opts, nil
}
