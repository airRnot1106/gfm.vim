# gfm.vim

Jump to any word in the window by typing a few letters of it, in Vim and Neovim.

gfm.vim is a port of [fuzzy-motion.vim](https://github.com/yuki-yano/fuzzy-motion.vim) from [denops.vim](https://github.com/vim-denops/denops.vim) to [goda.vim](https://github.com/Shougo/goda.vim). It keeps `:FuzzyMotion`, the labels, the highlights and the `g:fuzzy_motion_*` options, and needs no Deno.

## Features

- Works in Vim and Neovim
- The `fzf` matcher is a port of [fzf-for-js](https://github.com/ajitid/fzf-for-js), the extended syntax included
- The `kensaku` matcher finds Japanese words from romaji with the Go package of [gks.vim](https://github.com/airRnot1106/gks.vim), linked into the binary

## Requirements

- Vim 9.1 with `+job` and `+channel`, or Neovim 0.11
- [goda.vim](https://github.com/Shougo/goda.vim)
- [gks.vim](https://github.com/airRnot1106/gks.vim), for the `kensaku` matcher
- [Nix](https://nixos.org) with flakes enabled, or [Go](https://go.dev) 1.26, to build the binary

Windows is not supported.

## Installation

### Using a plugin manager

Install goda.vim, gks.vim and gfm.vim, then run `nix build` inside the plugin directories. The plugin expects the binary at `result/bin/gfm`.

With [lazy.nvim](https://github.com/folke/lazy.nvim):

```lua
{
  'airRnot1106/gfm.vim',
  dependencies = { 'Shougo/goda.vim', 'airRnot1106/gks.vim' },
  build = 'nix build',
  cmd = 'FuzzyMotion',
  init = function()
    vim.g.fuzzy_motion_matchers = { 'kensaku', 'fzf' }
  end,
}
```

### Without Nix

Build the binary with Go instead. Run this inside the plugin directory:

```sh
go build -o bin/gfm .
```

The plugin finds the binary at `bin/gfm`.

## Usage

`:FuzzyMotion`, then the search in lowercase, then a label in uppercase. `<CR>` jumps to the first target.

```vim
nnoremap <Leader>; <Cmd>FuzzyMotion<CR>
xnoremap <Leader>; <Cmd>FuzzyMotion<CR>
```

See `:help gfm`.

## Options

| Variable | Default | Description |
| --- | --- | --- |
| `g:gfm_binary` | Unset | Path to the binary. Overrides `bin/gfm` and `result/bin/gfm`. |
| `g:fuzzy_motion_labels` | `A` to `Z`, home row first | The labels. |
| `g:fuzzy_motion_matchers` | `['fzf']` | `fzf` and `kensaku`. |
| `g:fuzzy_motion_word_regexp_list` | See `:help` | Regular expressions that make the words. |
| `g:fuzzy_motion_word_filter_regexp_list` | `[]` | Words have to match all of these. |
| `g:fuzzy_motion_auto_jump` | `v:false` | Jump when one target is left. |
| `g:fuzzy_motion_disable_match_highlight` | `v:false` | Do not highlight matches. |

The highlight groups are `FuzzyMotionShade`, `FuzzyMotionChar`, `FuzzyMotionSubChar` and `FuzzyMotionMatch`. If something fails, check `goda#log('gfm')`.

## Differences from fuzzy-motion.vim

- The regular expression options use the syntax of Go's `regexp` package instead of JavaScript's. Look-around and backreferences are not available. `\p{Script_Extensions=Latin}` works, and other `Script_Extensions` properties are read as the plain script.
- The offsets of `fzf` results are bytes, so a label lands on the matched character in a word with multibyte text before it.
- A search with no position, such as `^` or `!foo`, labels the start of each word, and `'` in the search works. fuzzy-motion.vim stops with an error on `^` at the start of a line and on `'`.
- The shade covers the lines of the window. fuzzy-motion.vim also adds a match for each line up to the sum of the first and last line numbers, which delays the labels in the middle of a long buffer.
- Decomposed Unicode text is not normalized before `fzf` matches it.
- `fuzzy_motion#request()` is gone. `fuzzy_motion#targets()` stays.

## Development

```sh
nix develop
go test ./...
nix flake check
```

`internal/motion/testdata/targets.json` is what `fuzzy_motion#targets()` of fuzzy-motion.vim returned for 80 queries on `fixture.txt`, and `internal/fzf/testdata/golden.ts` writes the results of fzf 0.5.2. The kensaku cases read the dictionary from `$GKS_TEST_DICT` or `~/.cache/kensaku.vim/migemo-compact-dict`, and skip without it.

## Special thanks

- [fuzzy-motion.vim](https://github.com/yuki-yano/fuzzy-motion.vim)
- [fzf-for-js](https://github.com/ajitid/fzf-for-js)
- [fzf](https://github.com/junegunn/fzf)

## License

MIT, except `internal/fzf`, which is under the BSD 3-Clause License of fzf-for-js.
