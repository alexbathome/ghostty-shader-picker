package picker

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"syscall"

	"github.com/alexbathome/ghostty-shader-picker/internal/ui"
)

const (
	name            = "ghostty-shader-picker"
	configTemplate  = "custom-shader = %q"
	ghosttyProcName = "ghostty"
)

var (
	//go:embed ghostty-shaders-dist/*.glsl
	inbuiltShaders embed.FS
	//go:embed synopsis.txt
	synopsis string
)

// Main is the main entry point for the picker.
func Main(_ context.Context) error {
	var (
		fs = NewCustomFlagSet(name, flag.ExitOnError)

		inputShaderDirs       = fs.StringSlice("shader-dir", []string{}, "directory to load shaders from instead of inbuilt ones (optional)")
		inputShaderFiles      = fs.StringSlice("shader-file", []string{}, "individual shader file to load instead of inbuilt ones (optional)")
		includeInbuiltShaders = fs.Bool("include-inbuilt-shaders", true, "include inbuilt shaders in the list of shaders to pick from")
		autoApplyConfig       = fs.Bool("apply", false, "automatically reload ghostty's config with the picked shader (optional) (requires execution inside Ghostty)")
		version               = fs.Bool("version", false, "print the version of ghostty-shader-picker")
	)
	if err := fs.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	if *version {
		fmt.Println(getVersion())
		return nil
	}

	shaders, err := collectShaders(*inputShaderDirs, *inputShaderFiles, *includeInbuiltShaders)
	if err != nil {
		return fmt.Errorf("collecting shaders: %w", err)
	}

	pick, err := ui.Pick(shaders)
	if err != nil {
		return fmt.Errorf("picking a shader: %w", err)
	}
	if pick == nil {
		// pick is empty when the user has used the 'q' quit.
		return nil
	}

	outputShaderPath, err := installShader(*pick)
	if err != nil {
		return fmt.Errorf("installing shader: %w", err)
	}

	configPath, err := PreferredConfigDir()
	if err != nil {
		return fmt.Errorf("getting preferred config dir: %w", err)
	}
	configPath = filepath.Join(configPath, "auto", "shader.ghostty")

	if err := updateGhosttyAutoConfig(configPath, outputShaderPath); err != nil {
		return fmt.Errorf("updating ghostty auto config: %w", err)
	}

	if *autoApplyConfig {
		if err := refreshGhostty(); err != nil {
			return fmt.Errorf("updating ghostty config: %w", err)
		}
	}

	return nil
}

// collectShaders collects shaders from the specified directories and files, as
// well as the inbuilt shaders if includeInbuiltShaders is true. It returns a
// list of ShaderModel representing the available shaders to pick from.
func collectShaders(dirs, files []string, includeInbuiltShaders bool) ([]ui.ShaderModel, error) {
	shaders := []ui.ShaderModel{}

	if includeInbuiltShaders {
		inbuilt, err := inbuiltShaders.ReadDir("ghostty-shaders-dist")
		if err != nil {
			return nil, fmt.Errorf("reading inbuilt shaders: %w", err)
		}
		for _, file := range inbuilt {
			shaders = append(shaders, ui.ShaderModel{Name: file.Name(), Meta: "(inbuilt)", Builtin: true})
		}
	}

	for _, dir := range dirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("reading shader dir %q: %w", dir, err)
		}
		for _, file := range files {
			if !file.IsDir() {
				shaders = append(shaders, ui.ShaderModel{Name: file.Name(), Meta: fmt.Sprintf("(from: %s)", dir), Builtin: false})
			}
		}
	}

	for _, file := range files {
		shaders = append(shaders, ui.ShaderModel{Name: file, Meta: "(specified)", Builtin: false})
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

func installShader(pick ui.ShaderModel) (string, error) {
	ucd, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("getting user cache dir: %w", err)
	}

	cacheDir := filepath.Join(ucd, name)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("creating cache dir: %w", err)
	}

	var (
		pickPath = pick.Name
		src      io.ReadCloser
		readErr  error
	)
	if pick.Builtin {
		pickPath = filepath.Join("ghostty-shaders-dist", pick.Name)
		src, readErr = inbuiltShaders.Open(pickPath)
	} else {
		src, readErr = os.Open(pickPath)
	}
	if readErr != nil {
		return "", fmt.Errorf("opening shader file: %w", readErr)
	}
	defer src.Close()

	dst := filepath.Join(cacheDir, "shader.glsl")
	out, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("creating shader file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return "", fmt.Errorf("writing shader file: %w", err)
	}
	return dst, nil
}

// Updates the {ghostty config path}/auto/shader.ghostty file with the provided
// shader path.
func updateGhosttyAutoConfig(configPath, shaderPath string) error {
	template := fmt.Appendf([]byte{}, "custom-shader = %q", shaderPath)
	err := os.WriteFile(configPath, template, 0644)
	if err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// getVersion returns the version of ghostty-shader-picker, or "unknown" if it
// cannot be determined.
func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "unknown"
}
