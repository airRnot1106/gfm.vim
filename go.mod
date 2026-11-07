module github.com/airRnot1106/gfm.vim

go 1.26

require (
	github.com/Shougo/goda.vim v0.7.2
	github.com/airRnot1106/gks.vim v0.0.0
)

// gks.vim is not published yet.
replace github.com/airRnot1106/gks.vim => ../gks.vim
