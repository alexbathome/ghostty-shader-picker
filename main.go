package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alexbathome/ghostty-shader-picker/internal/picker"
)

func main() {
	if err := picker.Main(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}	
}