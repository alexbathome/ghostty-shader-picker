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

// Set implements the flag.Value interface. It appends the provided value to
// the slice,
func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// cfs is a custom FlagSet that provides additional flag types.
// and custom usage output.
type cfs struct {
	*flag.FlagSet
}

// NewCustomFlagSet creates a new custom FlagSet with the specified name and
// error handling behavior. It also sets up a custom usage function that
// formats the usage output according to the synopsis string.
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

// StringSlice adds a flag that can be repeated multiple times to build up a
// slice of strings. For example:
//
//	--shader-dir dir1 --shader-dir dir2 --shader-file file1 --shader-file file2
//	// would result in:
//	[]string{"dir1", "dir2"} // for the shader-dir flag
//	// and:
//	[]string{"file1", "file2"} // for the shader-file flag.
func (c *cfs) StringSlice(name string, defaultValue []string, usage string) *stringSlice {
	var ss stringSlice = defaultValue
	c.Var(&ss, name, usage)
	return &ss
}
