# exex

A terminal UI for exploring **ELF, Mach-O and PE** binaries: file header, sections,
symbols, disassembly, hex/raw dumps, strings, dynamic libraries, and
DWARF-driven source mapping — all keyboard- and mouse-driven.

```
exex [-debug PATH] [-s STRING] <binary> [goto]
```

## Full vs lite build

There are two builds. They are identical except for syntax highlighting:

| Build | Size (stripped) | Syntax highlighting |
|-------|-----------------|---------------------|
| **full** | ~11 MB | [Chroma](https://github.com/alecthomas/chroma) — full multi-language source highlighting and assembly highlighting |
| **lite** | ~7 MB | a small built-in highlighter (categorized source keywords/function names; categorized asm mnemonics plus registers / immediates / links) |

The lite build drops Chroma and its ~3 MB of embedded lexer/style data. Both
builds honour the same themes and `colors:` config; the built-in highlighter
follows your theme too.

Pick **lite** if you want the smaller binary, **full** for the richest colouring.

## Install

### From a release

Download the asset for your OS/arch from the
[Releases](../../releases) page — add `-lite` to the name for the small build.

```sh
# macOS / Linux
tar -xzf exex-<version>-<os>-<arch>.tar.gz        # or ...-<arch>-lite.tar.gz
chmod +x exex
sudo mv exex /usr/local/bin/

# verify (optional)
shasum -a 256 -c checksums.txt
```

### With `go install`

```sh
go install github.com/rabarbra/exex@latest          # full build
go install -tags lite github.com/rabarbra/exex@latest   # lite build
```

### Build from source

```sh
make build    # full  -> ./exex
make lite     # lite  -> ./exex
make test     # go test + lite vet
```

## Usage

```
exex [flags] <binary> [goto]
```

- `<binary>` — path to an ELF/Mach-O/PE file, or a command name found on `$PATH`
  (e.g. `exex ls` opens `/bin/ls`).
- `goto` — optional address (`0x401000`) or symbol name to jump to on open. A
  unique symbol jumps straight to it; an ambiguous one opens the Symbols view
  filtered by it.

Flags (accepted in any position):

| Flag | Description |
|------|-------------|
| `-s STRING` | search printable strings; opens the match in Hex, or the Strings view filtered when several match |
| `-debug PATH` / `-d PATH` | external debug-symbols file or directory (ELF `.debug` companion, or a Mach-O `.dSYM` bundle/file) |

### Keys

| Key | Action |
|-----|--------|
| `1`–`9` | switch view (Info, Sections, Symbols, Disasm, Hex, Libs, Raw, Strings, Sources) |
| `↑/↓` `j/k`, `PgUp/PgDn`, `Home/End` | move / page (also `⌘↑`/`⌘↓`, `^A`/`^E` on macOS) |
| `/` | filter / search the current view |
| `Enter` | open / follow / jump |
| `g` | go to address or symbol |
| `[` / `]` | previous / next (section in Hex/Raw, symbol in Disasm) |
| `⇧[` / `⇧]` | previous / next non-zero byte (Hex/Raw) |
| `d` | disassemble selected address (when executable) |
| `a` / `s` | copy address / name |
| `w` | toggle long-line wrap |
| `Tab` / `⇧Tab` | show-hide / swap the disasm source pane |
| `?` | full key reference · `q` quit |

The mouse wheel scrolls, click selects, and double-click follows in the disasm view.

## Configuration

Config is optional YAML at:

```
$XDG_CONFIG_HOME/exex/config.yaml      # or, if XDG_CONFIG_HOME is unset:
$HOME/.config/exex/config.yaml
```

Every field is optional — unset entries keep their defaults, so you only specify
what you want to change. You can:

- pick a built-in **theme** preset: `theme: dark | nord | solarized-dark | solarized-light`,
- override any individual colour under `colors:` (instruction classes, address
  links, tables, source/asm highlight, the hex byte ramp, path colours, …),
- rebind top-level **keys**,
- set **behaviour** (default view, disasm landing target, decode window size).

See [`docs/config.example.yaml`](docs/config.example.yaml) for the full annotated
schema. Colour values are a `#RRGGBB` hex string or an ANSI-256 index (e.g.
`"203"`).
