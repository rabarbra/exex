# Ideas — UX & exploration

The north star: open *anything* executable (or library / object file) and easily
explore it — see what it contains and how it's organized, find any symbol /
address / string and jump to it immediately, see and explore dependencies, and
see what it *uses* and *needs* (syscalls, CPU features, arch/OS).

Below are UX improvements ranked by how much they make working with exex clearer
and more straightforward. (Feature-level capability work — CPU features,
requirements panel, dyld cache — lives in `docs/ROADMAP.md` #34–#37.)

---

## Tier 1 — biggest clarity wins

### 1. Universal "jump to anything" palette  ⭐ (in progress)

Today finding things is split across `g` (goto: symbol/addr), `/` (in-view
search) and five per-view filters — the user must know which to use. One key
should open a fuzzy palette over **everything** and jump on Enter.

- **Scopes** (switchable, e.g. Tab to cycle, shown as a segmented bar):
  - **All** — symbols + sections + libraries (+ auto-detect a typed address).
  - **Symbols**, **Sections**, **Strings**, **Libraries**, **Address**.
  - Strings is its own scope (the corpus can be millions — don't scan it in All
    on every keystroke).
- **Address mode** — interpret the typed address as **virtual** (default) or
  **physical / LMA**. Physical input resolves through the section whose LMA range
  contains it (`virtual = sec.Addr + (input − sec.PhysAddr)`), so a higher-half
  kernel can be navigated by physical address. Only offered when the binary has
  distinct LMAs. (Synthetic-address objects: real position is section-relative,
  so "physical" doesn't apply there.)
- **Destination is smart by result kind** — symbol → disasm (or hex if not
  code), section → hex at its address, string → hex/strings, library → open it,
  address → disasm/hex/raw by mapping. (Explicit "go to in <view>" picking is a
  possible later refinement — keep the default smart for now.)
- **Results coloured by kind** (matching the Symbols view + the xref/syscall
  modal vocabulary), with a kind tag.

### 2. Cross-file back-stack + breadcrumb

`openLibAsPrimary` / `loadArchiveMember` / `switchFatArch` return fresh models
with **no way back** — opening a dependency is a one-way trip, which makes
"explore dependencies" disorienting.

- Keep a small stack of opened files (path + the view/cursor you left).
- Show a breadcrumb in the header: `/bin/app ▸ libfoo.so ▸ kbd_write`.
- Esc / a back key pops to where you came from.

### 3. Info as a landing dashboard

It's the first screen but a flat field list. Make it answer at a glance:

- **What is this** — format / arch / type.
- **What it needs** — arch + bits + endianness, OS/ABI, PIE/static/dynamic, CPU
  baseline (ties to ROADMAP #34/#35).
- **What's inside** — symbol / section / string / library counts, **each with an
  inline "→ press N" jump hint** (today only libs has one).
- **Entry point** and **top dependencies**.

---

## Tier 2 — clarity polish

### 4. Make the search model legible

`/` (search within view) vs `g`/palette (jump) vs per-view filters overlap. Show
the active filter/scope as a persistent chip in the header (`filter: "kbd" ·
12/138`) so it's always clear what's narrowing the view and how to clear it.

### 5. Colour / vocabulary legend in `?`

A lot of colour now carries meaning (symbol kinds, section categories,
syscall/xref categories) plus address terms (synthetic / LMA). A short legend
reachable from help removes "why is this row yellow / what's LMA?".

### 6. Direct mouse affordances

Click a library row to open it; click a symbol / xref / goto result to jump
(some exists). Lower the barrier for mouse-first users.

---

## Tier 3 — polish

- First-run hint line (`? help · g jump · 1–9 views`).
- Clearer `t`-toggle affordance — persistently show what `t` does in the current
  view (it's heavily context-overloaded).
