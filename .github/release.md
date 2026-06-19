A terminal UI for exploring ELF, Mach-O and PE binaries — header, sections, symbols, disassembly, hex, strings, libraries and DWARF-driven source mapping.

### Which build do I want?

| Build | Size | Syntax highlighting |
|-------|------|---------------------|
| **full** (`exex-…-<os>-<arch>.tar.gz`) | ~11 MB | Chroma — full multi-language source + asm highlighting |
| **lite** (`exex-…-<os>-<arch>-lite.tar.gz`) | ~7 MB | built-in minimal highlighter |

Everything else is identical, and both honour the same themes/colours. Pick **lite** for the smaller binary, **full** for the richest colouring.

### Install (macOS / Linux)

```sh
tar -xzf exex-<version>-<os>-<arch>.tar.gz   # add -lite for the small build
chmod +x exex && sudo mv exex /usr/local/bin/
shasum -a 256 -c checksums.txt          # optional: verify
```

### Usage

```
exex [-debug PATH] [-s STRING] <binary> [goto]
```

Config lives at `$XDG_CONFIG_HOME/exex/config.yaml` (or `~/.config/exex/config.yaml`). The bundled `README.md` and `config.example.yaml` document the keys, flags and full colour/theme schema.
