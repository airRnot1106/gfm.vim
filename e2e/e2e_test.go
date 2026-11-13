// Package e2e runs :FuzzyMotion in real editors.
package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var editors = []struct {
	name    string
	command []string
}{
	{"vim", []string{"vim", "-u", "NONE", "-N", "-n", "-i", "NONE", "-es", "-S"}},
	{"nvim", []string{"nvim", "-u", "NONE", "-n", "-i", "NONE", "--headless", "-S"}},
}

func moduleDir(t *testing.T, root, env, module string) string {
	t.Helper()
	if dir := os.Getenv(env); dir != "" {
		return dir
	}
	list := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", module)
	list.Dir = root
	out, err := list.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func dictionary(t *testing.T) string {
	t.Helper()
	path := os.Getenv("GODA_KS_TEST_DICT")
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".cache/kensaku.vim/migemo-compact-dict")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skip("no dictionary: ", err)
	}
	return path
}

func TestEditors(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "goda-fm")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	for _, editor := range editors {
		t.Run(editor.name, func(t *testing.T) {
			if _, err := exec.LookPath(editor.command[0]); err != nil {
				if os.Getenv("GODA_FM_E2E_STRICT") != "" {
					t.Fatal(err)
				}
				t.Skip(err)
			}
			out := filepath.Join(t.TempDir(), "out.json")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, editor.command[0], append(editor.command[1:], "e2e/testdata/driver.vim")...)
			cmd.Dir = root
			cmd.Env = append(os.Environ(),
				"GODA_FM_E2E_PLUGIN="+root,
				"GODA_FM_E2E_GODA="+moduleDir(t, root, "GODA_FM_E2E_GODA", "github.com/Shougo/goda.vim"),
				"GODA_FM_E2E_GODA_KS="+moduleDir(t, root, "GODA_FM_E2E_GODA_KS", "github.com/airRnot1106/goda-ks.vim"),
				"GODA_FM_E2E_BINARY="+binary,
				"GODA_FM_E2E_DICT="+dictionary(t),
				"GODA_FM_E2E_OUT="+out,
			)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v\n%s", err, output)
			}
			data, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{
				"english":   []any{3.0, 7.0},
				"romaji":    []any{2.0, 8.0},
				"label":     []any{3.0, 1.0},
				"cancel":    []any{1.0, 1.0},
				"backspace": []any{2.0, 18.0},
				"matches":   0.0,
				"auto_jump": []any{2.0, 1.0},
				"targets":   1.0,
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %v\nwant %v", got, want)
			}
		})
	}
}
