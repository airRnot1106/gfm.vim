let s:root = expand('<sfile>:p:h:h')
let s:namespace = has('nvim') ? nvim_create_namespace('fuzzy-motion') : 0
let s:prop_type = 'fuzzy-motion'
let s:prop_id = 0
let s:marks = []
let s:popups = []
let s:matches = []

let s:ENTER = 13
let s:ESC = 27
let s:BS = 128
let s:C_H = 8
let s:C_W = 23
let s:SPACE = 32
let s:TILDE = 126

function! s:start() abort
  let bins = exists('g:gfm_binary') ? [g:gfm_binary] : []
  let bins += [s:root . '/bin/gfm', s:root . '/result/bin/gfm']
  for bin in bins
    if executable(bin)
      call goda#start(bin)
      return
    endif
  endfor
  throw 'gfm: no binary found, run nix build in ' . s:root . ' or see :help gfm-install'
endfunction

" Starts the binary, and with it the load of the dictionary while the user types.
" goda keeps a failed start until the next goda#register(), and a later jump has
" to see a binary built in the meantime.
function! s:prepare() abort
  if !goda#is_running('gfm')
    call goda#register('gfm', function('s:start'),
          \ #{version: 1, module: 'github.com/airRnot1106/gfm.vim'})
  endif
  let config = index(g:fuzzy_motion_matchers, 'kensaku') >= 0 ? kensaku#config() : {}
  call goda#notify('gfm', 'configure', config)
endfunction

function! s:request(input, previous, first, lines) abort
  return goda#request('gfm', 'targets', #{
        \ input: a:input,
        \ previous: a:previous,
        \ first: a:first,
        \ lines: a:lines,
        \ regexps: g:fuzzy_motion_word_regexp_list,
        \ filters: g:fuzzy_motion_word_filter_regexp_list,
        \ labels: g:fuzzy_motion_labels,
        \ matchers: g:fuzzy_motion_matchers,
        \})
endfunction

function! fuzzy_motion#targets(q) abort
  call s:prepare()
  let first = line('w0')
  return s:request(a:q, [], first, getline(first, line('w$')))
endfunction

function! fuzzy_motion#_getchar() abort
  let l:code = getchar()
  if type(l:code) == v:t_string
    let l:code = char2nr(l:code)
  endif
  return l:code
endfunction

" The label covers the character before the match, so that the match stays
" readable, and takes that character's display width to keep the text in place.
function! s:mount_label(target, highlight) abort
  let col = a:target.pos.col + a:target.start
  if !has('nvim')
    let s:prop_id += 1
    call add(s:marks, s:prop_id)
    call prop_add(a:target.pos.line, col, #{type: s:prop_type, id: s:prop_id})
    call add(s:popups, popup_create(a:target.char, #{
          \ line: -1,
          \ col: -1,
          \ textprop: s:prop_type,
          \ textpropid: s:prop_id,
          \ width: 1,
          \ height: 1,
          \ highlight: a:highlight,
          \}))
    return
  endif
  let line = getline(a:target.pos.line)
  let index = charidx(line, col - 1)
  let base = strcharpart(line, max([index - 1, 0]), 1, 1)
  let start = index == 0 ? col - 1 : col - 1 - len(base)
  let padding = repeat(' ', strdisplaywidth(base) - strchars(a:target.char))
  call nvim_buf_set_extmark(0, s:namespace, a:target.pos.line - 1, start, #{
        \ virt_text: [[padding . a:target.char, a:highlight]],
        \ virt_text_pos: 'overlay',
        \ hl_mode: 'combine',
        \})
endfunction

function! s:mount(targets) abort
  let highlight = 'FuzzyMotionChar'
  for target in a:targets
    call s:mount_label(target, highlight)
    let highlight = 'FuzzyMotionSubChar'
    if !g:fuzzy_motion_disable_match_highlight && target.end > target.start
      call add(s:matches, matchaddpos('FuzzyMotionMatch',
            \ [[target.pos.line, target.pos.col + target.start, target.end - target.start]], 20))
    endif
  endfor
endfunction

function! s:unmount() abort
  if has('nvim')
    call nvim_buf_clear_namespace(0, s:namespace, 0, -1)
  else
    for id in s:marks
      call prop_remove(#{type: s:prop_type, id: id})
    endfor
    call map(s:popups, 'popup_close(v:val)')
    let s:popups = []
    let s:marks = []
  endif
  call map(s:matches, 'matchdelete(v:val)')
  let s:matches = []
endfunction

function! s:jump(target) abort
  normal! m`
  call cursor(a:target.pos.line, a:target.pos.col + a:target.start)
endfunction

function! s:find_label(targets, code) abort
  for target in a:targets
    if char2nr(target.char) == a:code
      return target
    endif
  endfor
  return v:null
endfunction

function! s:loop(first, lines) abort
  let labels = map(copy(g:fuzzy_motion_labels), 'char2nr(v:val)')
  let input = ''
  let targets = []
  while 1
    echo 'fuzzy-motion: ' . input
    call s:unmount()
    call s:mount(targets)
    redraw

    let code = fuzzy_motion#_getchar()
    if code == s:ENTER
      if empty(targets)
        return
      endif
      let code = char2nr(targets[0].char)
    endif

    if code == s:ESC
      return
    elseif index(labels, code) >= 0
      let target = s:find_label(targets, code)
      if target isnot v:null
        call s:jump(target)
        return
      endif
    elseif code == s:BS || code == s:C_H
      let input = input[: -2]
      let targets = s:request(input, [], a:first, a:lines)
    elseif code == s:C_W
      let input = ''
      let targets = []
    elseif s:SPACE <= code && code <= s:TILDE
      let input .= nr2char(code)
      let targets = s:request(input, targets, a:first, a:lines)
      if g:fuzzy_motion_auto_jump && len(targets) == 1
        call s:jump(targets[0])
        return
      endif
    endif
  endwhile
endfunction

function! fuzzy_motion#execute() abort
  if !has('nvim') && empty(prop_type_get(s:prop_type))
    call prop_type_add(s:prop_type, {})
  endif
  let first = line('w0')
  let last = line('w$')
  let shade = map(range(first, last), {_, line -> matchaddpos('FuzzyMotionShade', [line], 10)})
  try
    echo 'fuzzy-motion: '
    redraw
    call s:prepare()
    call s:loop(first, getline(first, last))
  catch /^Vim:Interrupt$/
  catch
    echohl ErrorMsg
    echomsg substitute(v:exception, '^Vim(\a\+):Invoking .\{-} on channel \d\+:\n', '', '')
    echohl None
  finally
    call map(shade, 'matchdelete(v:val)')
    call s:unmount()
    echo ''
    execute has('nvim') ? 'redraw' : 'redraw!'
  endtry
endfunction
