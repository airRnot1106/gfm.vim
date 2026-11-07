// Package motion is the gfm plugin. It finds and labels the targets of
// an input for autoload/fuzzy_motion.vim.
package motion

import (
	"context"
	"slices"

	"github.com/Shougo/goda.vim/sdk"

	"github.com/airRnot1106/gks.vim/migemo"
)

// scriptVersion matches the version autoload/fuzzy_motion.vim passes to
// goda#register.
const scriptVersion = 1

// Motion holds no state of a jump: every request carries it.
type Motion struct {
	loader migemo.Loader
}

// New returns a plugin with no dictionary configured.
func New() *Motion {
	return &Motion{}
}

// Name implements sdk.Plugin.
func (*Motion) Name() string { return "gfm" }

// ScriptVersion implements sdk.ScriptVersioner.
func (*Motion) ScriptVersion() int { return scriptVersion }

// Handlers implements sdk.Plugin.
func (p *Motion) Handlers() map[string]sdk.Handler {
	return map[string]sdk.Handler{
		"configure": p.configure,
		"targets":   p.targets,
	}
}

func (p *Motion) configure(_ context.Context, _ sdk.Host, params []any) (any, error) {
	var config migemo.Config
	if err := sdk.DecodeArgs(params, &config); err != nil {
		return nil, err
	}
	if config.Dictionary != "" {
		p.loader.Configure(config)
	}
	return nil, nil
}

func (p *Motion) targets(_ context.Context, _ sdk.Host, params []any) (any, error) {
	var r Request
	if err := sdk.DecodeArgs(params, &r); err != nil {
		return nil, err
	}
	var m *migemo.Migemo
	if slices.Contains(r.Matchers, "kensaku") {
		var err error
		if m, err = p.loader.Migemo(); err != nil {
			return nil, err
		}
	}
	found, err := targets(r, m)
	if err != nil {
		return nil, err
	}
	return sdk.Generic(found)
}
