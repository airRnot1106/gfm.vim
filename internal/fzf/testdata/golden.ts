// Writes golden.json from fzf 0.5.2, the matcher of fuzzy-motion.vim.
//
//   deno run -A internal/fzf/testdata/golden.ts > internal/fzf/testdata/golden.json
import { extendedMatch, Fzf } from "npm:fzf@0.5.2";

const texts = [
  "foo", "foobar", "foo_bar", "fooBarBaz", "FOO", "Foo.Bar", "foo-bar-baz", "barfoo", "f_o_o", "fxoxo",
  "src/main.go", "internal/fzf/algo.go", "README.md", "a.b.c", "a", "ab", "abc", "abcabc", "aXbXc", "camelCaseWord",
  "snake_case_word", "kebab-case-word", "v1.2.3", "item1", "item2", "ITEM3", "x1y2z3", "(paren)", "#tag", "foo bar",
  "  leading", "trailing  ", " both ", "日本語", "日本語 abc", "abc 日本語 def", "カタカナabc", "café", "naïve", "résumé",
  "Ångström", "ÀÉÎÕÜ", "ßtraße", "😀 smile", "a😀b", "tab\tseparated", "ﾊﾝｶｸ 123", "ｚｅｎ ｋａｋｕ", "ÇA", "çà",
  "a".repeat(40000) + "b" + "c".repeat(30000),
];

const queries = [
  "f", "fo", "foo", "fb", "fbb", "o", "oo", "bar", "Bar", "BAR", "FOO", "ab", "abc", "ac", "acb", "cw", "ccw", "sc",
  "1", "12", "123", "item", "i1", "main", "go", "a.b", "(", "#", "e", "cafe", "naive", "resume", "angstrom", "a", "aeiou",
  "é", "日本", "abc 日", "'foo", "'oo", "'bar", "'ab", "^foo", "^a", "^ ", "^bar", "foo$", "bar$", "c$", "word$", "^foo$", "^abc$",
  "^a$", "!foo", "!a", "!foo !bar", "!^f", "!o$", "foo bar", "f b", "foo | bar", "foo | bar | abc", "a | b c", "'foo | ^bar",
  "foo\\ bar", "\\ ", "|", "| a", "a |", "$", "^", "'", "!", "!'foo", "'foo$", "^'x", "abc abc", "aaa", "ab$", "ACB",
];

const fzf = new Fzf(texts, { selector: (s: string) => s, match: extendedMatch });
const out = queries.map((query) => ({
  query,
  results: fzf.find(query).map((r) => [texts.indexOf(r.item), r.start, r.end, r.score]),
}));
console.log(JSON.stringify({ texts, queries: out }));
