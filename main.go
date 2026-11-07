package main

import (
	"github.com/Shougo/goda.vim/host"

	"github.com/airRnot1106/gfm.vim/internal/motion"
)

func main() {
	host.Main(motion.New())
}
