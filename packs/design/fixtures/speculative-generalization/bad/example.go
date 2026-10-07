package bad

type Options struct {
	Prefix  string
	MaxLen  int
	Verbose bool
}

func Format(name string, opts Options) string {
	return opts.Prefix + name
}
