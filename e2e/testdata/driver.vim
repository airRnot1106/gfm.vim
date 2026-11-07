" Runs :FuzzyMotion in a real editor and writes what it saw to $GFM_E2E_OUT.
set encoding=utf-8
let &runtimepath = join(map([$GFM_E2E_PLUGIN, $GFM_E2E_GODA, $GFM_E2E_GKS], {_, dir -> escape(dir, ',')}) + [&runtimepath], ',')
let g:gfm_binary = $GFM_E2E_BINARY
let g:kensaku_dictionary_cache = $GFM_E2E_DICT
let g:fuzzy_motion_matchers = ['kensaku', 'fzf']
runtime plugin/kensaku.vim
runtime plugin/fuzzy_motion.vim

let s:result = {}

function! s:jump(keys) abort
  call cursor(1, 1)
  call feedkeys(":FuzzyMotion\<CR>" . a:keys, 'tx')
  return [line('.'), col('.')]
endfunction

try
  call setline(1, ['first line', 'second 日本語 third', 'alpha beta'])
  let s:result.english = s:jump("beta\<CR>")
  let s:result.romaji = s:jump("nihongo\<CR>")
  let s:result.label = s:jump("alA")
  let s:result.cancel = s:jump("al\<Esc>")
  let s:result.backspace = s:jump("xyzzy\<BS>\<BS>\<BS>\<BS>\<BS>thi\<CR>")
  let s:result.matches = len(getmatches())

  let g:fuzzy_motion_auto_jump = v:true
  let s:result.auto_jump = s:jump('seco')
  let g:fuzzy_motion_auto_jump = v:false

  let s:result.targets = len(fuzzy_motion#targets('nihongo'))
catch
  let s:result.error = v:exception
endtry

call writefile([json_encode(s:result)], $GFM_E2E_OUT)
qall!
