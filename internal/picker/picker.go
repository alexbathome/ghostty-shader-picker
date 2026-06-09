package picker

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/alexbathome/ghostty-shader-picker/internal/ui"
)

const (
	name            = "ghostty-shader-picker"
	configTemplate  = "custom-shader = %q"
	ghosttyProcName = "ghostty"
)

var (
	//go:embed ghostty-shaders/*.glsl
	inbuiltShaders embed.FS
	//go:embed synopsis.txt
	synopsis string

	// version is set at build time using
	// -ldflags="-X github.com/alexbathome/ghostty-shader-picker/internal/picker.version=1.2.3"
	version = "TODO(dev)"
)

func Main(_ context.Context) error {
	var (
		fs = NewCustomFlagSet(name, flag.ExitOnError)

		inputShaderDirs       = fs.StringSlice("shader-dir", []string{}, "directory to load shaders from instead of inbuilt ones (optional)")
		inputShaderFiles      = fs.StringSlice("shader-file", []string{}, "individual shader file to load instead of inbuilt ones (optional)")
		includeInbuiltShaders = fs.Bool("include-inbuilt-shaders", true, "include inbuilt shaders in the list of shaders to pick from")
		autoApplyConfig       = fs.Bool("apply", false, "automatically reload ghostty's config with the picked shader (optional) (requires execution inside Ghostty)")
	)
	if err := fs.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	shaders, err := collectShaders(*inputShaderDirs, *inputShaderFiles, *includeInbuiltShaders)
	if err != nil {
		return fmt.Errorf("collecting shaders: %w", err)
	}

	pick, err := ui.Pick(shaders)
	if err != nil {
		return fmt.Errorf("picking a shader: %w", err)
	}
	if pick == "" {
		return nil
	}

	outputShaderPath, err := installShader(pick)
	if err != nil {
		return fmt.Errorf("installing shader: %w", err)
	}

	// Update config
	if err := updateGhosttyAutoConfig(outputShaderPath); err != nil {
		return fmt.Errorf("updating ghostty auto config: %w", err)
	}

	if *autoApplyConfig {
		if err := refreshGhostty(); err != nil {
			return fmt.Errorf("updating ghostty config: %w", err)
		}
	}

	return nil
}

func collectShaders(dirs, files []string, includeInbuiltShaders bool) ([]string, error) {
	shaders := []string{}

	if includeInbuiltShaders {
		inbuilt, err := inbuiltShaders.ReadDir("ghostty-shaders")
		if err != nil {
			return nil, fmt.Errorf("reading inbuilt shaders: %w", err)
		}
		for _, file := range inbuilt {
			shaders = append(shaders, file.Name())
		}
	}

	for _, dir := range dirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("reading shader dir %q: %w", dir, err)
		}
		for _, file := range files {
			if !file.IsDir() {
				shaders = append(shaders, file.Name())
			}
		}
	}

	for _, file := range files {
		shaders = append(shaders, file)
	}

	return shaders, nil
}

// refreshGhostty sends a SIGUSR2 signal to the ghostty process to trigger a
// live reload of its configuration.
func refreshGhostty() error {
	ghosttyProc, err := findGhostty(os.Getppid())
	if err != nil {
		return fmt.Errorf("finding ghostty process: %w", err)
	}
	ghosttyProc.Signal(syscall.SIGUSR2)
	return nil
}

func installShader(pick string, inbuilt bool) (string, error) {
	ucd, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("getting user cache dir: %w", err)
	}

	cacheDir := filepath.Join(ucd, name)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("creating cache dir: %w", err)
	}

	pickPath := pick
	if inbuilt {
		pickPath = filepath.Join("ghostty-shaders", pick)
	}
	pickedShaderFile, err := inbuiltShaders.Open(pickPath)
	if err != nil {
		return "", fmt.Errorf("opening shader file: %w", err)
	}
	defer pickedShaderFile.Close()

	outputShaderFile, err := os.Create(filepath.Join(cacheDir, "shader.glsl"))
	if err != nil {
		return "", fmt.Errorf("creating shader file: %w", err)
	}
	defer outputShaderFile.Close()

	if _, err := io.Copy(outputShaderFile, pickedShaderFile); err != nil {
		return "", fmt.Errorf("writing shader file: %w", err)
	}
	return outputShaderFile.Name(), nil
}

func updateGhosttyAutoConfig(shaderPath string) error {
	preferredConfigDir, err := PreferredConfigDir()
	if err != nil {
		return fmt.Errorf("getting preferred config dir: %w", err)
	}
	config := filepath.Join(preferredConfigDir, "auto", "shader.ghostty")
	f, err := os.OpenFile(config, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("opening auto config file: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, configTemplate, shaderPath)
	if err != nil {
		return fmt.Errorf("writing auto config file: %w", err)
	}
	return nil
}
