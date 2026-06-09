package picker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexbathome/ghostty-shader-picker/internal/ui"
)

func TestInstallShader(t *testing.T) {
	exampleShaders := map[string][]byte{
		"foo.glsl": []byte("foo contents"),
		"bar.glsl": []byte("bar contents"),
	}

	tempDir := t.TempDir()
	for name, content := range exampleShaders {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("unexpected error writing example shader file: %v", err)
		}
	}

	testCases := []struct {
		desc        string
		haveShader  ui.ShaderModel
		wantContent string
	}{
		{
			desc: "basic builtin case",
			haveShader: ui.ShaderModel{
				Name:    filepath.Join(tempDir, "foo.glsl"),
				Meta:    "",
				Builtin: false,
			},
			wantContent: string(exampleShaders["foo.glsl"]),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			td := t.TempDir()
			t.Setenv("XDG_CACHE_DIR", td)
			dst, err := installShader(tC.haveShader)
			if err != nil {
				t.Fatalf("unexpected error installing shader: %v", err)
			}

			output, err := os.ReadFile(dst)
			if err != nil {
				t.Fatalf("unexpected error reading installed shader: %v", err)
			}
			if string(output) != tC.wantContent {
				t.Errorf("got installed shader content %q, want %q", string(output), tC.wantContent)
			}
		})
	}
}

func TestUpdateGhosttyAutoConfig(t *testing.T) {
	testCases := []struct {
		desc              string
		haveShaderPath    string
		wantConfigContent string
	}{
		{
			desc:              "basic case",
			haveShaderPath:    "foo",
			wantConfigContent: "custom-shader = \"foo\"",
		},
		{
			desc:              "another case",
			haveShaderPath:    "bar",
			wantConfigContent: "custom-shader = \"bar\"",
		},
		{
			desc:              "another case",
			haveShaderPath:    "/tmp/baz.glsl",
			wantConfigContent: "custom-shader = \"/tmp/baz.glsl\"",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "shader.ghostty")
			if err := updateGhosttyAutoConfig(out, tC.haveShaderPath); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			gotBytes, err := os.ReadFile(out)
			if err != nil {
				t.Fatalf("unexpected error reading config file: %v", err)
			}
			got := string(gotBytes)
			if got != tC.wantConfigContent {
				t.Errorf("got config content %q, want %q", got, tC.wantConfigContent)
			}
		})
	}
}
