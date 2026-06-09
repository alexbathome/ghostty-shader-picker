package picker

import (
	"bytes"
	"flag"
	"fmt"
	"os"
)

type stringSlice []string

var _ flag.Value = (*stringSlice)(nil)

// StringSlice implements the flag.Value interface, allowing it to be used as a
// command-line flag that can be specified multiple times to build up a list of
// strings.
func (s *stringSlice) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type cfs struct {
	*flag.FlagSet
}

func NewCustomFlagSet(name string, errorHandling flag.ErrorHandling) *cfs {
	fs := &cfs{flag.NewFlagSet(name, errorHandling)}

	flagBuf := bytes.NewBuffer(nil)
	fs.SetOutput(flagBuf)

	fs.Usage = func() {
		fs.PrintDefaults()
		// PrintDefaults writes to the flagBuf, it has to be called before
		// formatting the synopsis string.
		synopsis = fmt.Sprintf(synopsis, flagBuf.String())
		fmt.Fprintf(os.Stderr, "%s", synopsis)
	}

	return fs
}

func (c *cfs) StringSlice(name string, defaultValue []string, usage string) *stringSlice {
	var ss stringSlice = defaultValue
	c.Var(&ss, name, usage)
	return &ss
}
