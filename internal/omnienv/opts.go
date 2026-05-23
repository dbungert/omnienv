package omnienv

type Opts struct {
	Launch  bool   `long:"launch"            description:"Create environment"`
	System  string `long:"system"  short:"s" description:"Override system value"`
	Verbose bool   `long:"verbose" short:"v" description:"Increase logging verbosity"`
	Version bool   `long:"version"           description:"Show version"`
	Params  []string
}
