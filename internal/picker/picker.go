package picker

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/alexbathome/ghostty-shader-picker/internal/ui"
	"github.com/mitchellh/go-ps"
)

const ghosttyProcName = "ghostty"

//go:embed ghostty-shaders/*.glsl
var inbuiltShaders embed.FS

func Main(ctx context.Context) error {
	ghosttyProc, err := findGhostty(os.Getppid())
	if err != nil {
		return fmt.Errorf("finding ghostty: %w", err)
	}

	inbuilt, err := inbuiltShaders.ReadDir("ghostty-shaders")
	if err != nil {
		return fmt.Errorf("reading inbuilt shaders: %w", err)
	}

	shaders := []string{}
	for _, file := range inbuilt {
		shaders = append(shaders, file.Name())
	}

	// TODO: have a way to pass in another directory for shaders, or even a
	// single shader file
	pick, err := ui.Pick(shaders)
	if err != nil {
		return fmt.Errorf("picking a shader: %w", err)
	}

	outputShaderPath, err := installShader(pick)
	if err != nil {
		return fmt.Errorf("installing shader: %w", err)
	}

	// Update config
	if err := updateGhosttyAutoConfig(outputShaderPath); err != nil {
		return fmt.Errorf("updating ghostty auto config: %w", err)
	}

	ghosttyProc.Signal(syscall.SIGUSR2) // TODO: maybe we want to be more specific here and send a SIGHUP or something instead, but this is probably good enough for now
	return nil
}

func installShader(pick string) (string, error) {
	ucd, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("getting user cache dir: %w", err)
	}

	cacheDir := filepath.Join(ucd, "ghostty-shader-picker")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("creating cache dir: %w", err)
	}

	pickedShaderFile, err := inbuiltShaders.Open(filepath.Join("ghostty-shaders", pick))
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

	_, err = fmt.Fprintf(f, "custom-shader = %q", shaderPath)
	if err != nil {
		return fmt.Errorf("writing auto config file: %w", err)
	}
	return nil
}

func findGhostty(pid int) (*os.Process, error) {
	proc, err := ps.FindProcess(pid)
	switch {
	case err != nil:
		return nil, err
	case proc.Executable() == ghosttyProcName:
		return os.FindProcess(proc.Pid())
	case proc.PPid() == 0:
		return nil, fmt.Errorf("not found")
	}
	return findGhostty(proc.PPid())
}
