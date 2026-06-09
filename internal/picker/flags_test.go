package picker

import (
	"flag"
	"slices"
	"testing"
)

func TestStringSlice(t *testing.T) {
	testCases := []struct {
		desc string
		args []string
		want *stringSlice
	}{
		{
			desc: "empty",
			args: []string{},
			want: &stringSlice{},
		},
		{
			desc: "multiple values",
			args: []string{"--string-slice", "a", "--string-slice", "b", "--string-slice", "c"},
			want: &stringSlice{"a", "b", "c"},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			fs := NewCustomFlagSet("test", flag.ContinueOnError)
			got := fs.StringSlice("string-slice", []string{}, "a string slice flag")
			if err := fs.Parse(tC.args); err != nil {
				t.Fatalf("unexpected error parsing flags: %v", err)
			}
			if !slices.Equal(*got, *tC.want) {
				t.Errorf("got %v, want %v", *got, *tC.want)
			}
		})
	}
}
