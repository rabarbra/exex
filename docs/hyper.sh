#!/bin/sh
BIN=${1:-/bin/ls}
EXEX=${2:-exex}
echo "\n============"
hyperfine \
    --warmup 5 \
    "$EXEX '$BIN' -o strings" \
    "strings -a -t x '$BIN'"

echo "\n============"
hyperfine \
    --warmup 5 \
    "$EXEX '$BIN' -o syms" \
    "nm -C -n -a '$BIN'"

echo "\n============"
hyperfine --warmup 5 \
  "$EXEX '$BIN' -o sections" \
  "objdump -h '$BIN'"

echo "\n============"
hyperfine \
    "$EXEX '$BIN' -o disasm" \
    "objdump -d '$BIN'" \
    "otool -tvV '$BIN'"

echo "\n============"
 hyperfine \
    "$EXEX '$BIN' -o disasm-all" \
    "objdump -D '$BIN'"
