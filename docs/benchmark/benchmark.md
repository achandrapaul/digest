# Benchmarks

How fast digest is, measured in CI on every version bump you push, beside the release build. Each cell is time per operation · memory allocated per operation, so lower is better. A 60 fps frame is 16.7 ms.

- The release workflow records each run here in its own `chore: benchmark` commit.
- `mise run bench` prints the same tables locally without recording them; `mise run bench:full` adds 100k notes.
- History columns from different machines aren't directly comparable.

What runs, at 1, 100, 1k and 10k notes on disk:

- **Process**: starting the binary (`digest --help`) and its peak memory.
- **Startup and loading**: opening the dashboard (config, notes from disk, first frame), the heap it keeps, listing notes, reading local review state.
- **Redraw and navigation**: moving with j/k, the sync pulse, the header, opening a preview and stepping through previews.
- **Actions**: quick actions, a search keystroke, saving and deleting a note (each reloads only that note).
- **Screens and overlays**: archive, help, settings, brag view, review details, link menu, delete confirm, error.
- **Typing in a long note**: a keystroke in a 500, 2k and 10k-line note with the cursor at the top and at the bottom.

## Latest

v0.0.153 `921e69c` · 2026-10-11 12:35 UTC · Apple M1 (Virtual)

### Process

| Process start (`digest --help`) | Peak memory |
|---|---|
| 4.5 ms | 7.5 MB |

### Startup and loading

Columns are notes on disk (reviews for Load reviews, at most 1000).

| Scenario | 1 | 100 | 1000 | 10000 |
|---|---|---|---|---|
| Startup | 759 µs · 791 KB | 3.22 ms · 1.7 MB | 22.5 ms · 9.4 MB | 239 ms · 82.7 MB |
| Startup heap | 0.6 MB | 0.6 MB | 1.3 MB | 6.4 MB |
| Load notes | 110 µs · 17 KB | 3.43 ms · 1.2 MB | 38.5 ms · 12.0 MB | 735 ms · 119.9 MB |
| Load reviews | 41.2 µs · 6 KB | 4.1 ms · 529 KB | 38.3 ms · 5.4 MB | 39 ms · 5.4 MB |

### Redraw and navigation

Columns are notes on disk (reviews for Load reviews, at most 1000).

| Scenario | 1 | 100 | 1000 | 10000 |
|---|---|---|---|---|
| Navigate j/k | 1.42 ms · 314 KB | 1.29 ms · 354 KB | 1.57 ms · 641 KB | 30 ms · 8.7 MB |
| Pulse tick | 356 µs · 95 KB | 381 µs · 94 KB | 403 µs · 106 KB | 922 µs · 222 KB |
| Header | 38.7 µs · 11 KB | 39.9 µs · 12 KB | 69.8 µs · 12 KB | 344 µs · 12 KB |
| Preview open | 730 µs · 302 KB | 746 µs · 344 KB | 1.35 ms · 704 KB | 17.2 ms · 6.5 MB |
| Preview next | 887 µs · 334 KB | 1.35 ms · 639 KB | 1.74 ms · 1.1 MB | 5.36 ms · 6.5 MB |

### Actions

Columns are notes on disk (reviews for Load reviews, at most 1000).

| Scenario | 1 | 100 | 1000 | 10000 |
|---|---|---|---|---|
| Quick actions open | 1.09 ms · 294 KB | 1.01 ms · 318 KB | 1.81 ms · 533 KB | 31.6 ms · 7.8 MB |
| Quick actions move | 927 µs · 264 KB | 985 µs · 275 KB | 1.75 ms · 387 KB | 32.1 ms · 6.5 MB |
| Search keystroke | 909 µs · 371 KB | 1.17 ms · 419 KB | 1.4 ms · 479 KB | 3.71 ms · 1.0 MB |
| Save note | 1.42 ms · 251 KB | 1.33 ms · 280 KB | 2.44 ms · 509 KB | 28.8 ms · 6.9 MB |
| Delete note | 899 µs · 216 KB | 967 µs · 234 KB | 1.47 ms · 393 KB | 27.2 ms · 6.7 MB |

### Screens and overlays

Columns are notes on disk (reviews for Load reviews, at most 1000).

| Scenario | 1 | 100 | 1000 | 10000 |
|---|---|---|---|---|
| Archive | 351 µs · 149 KB | 359 µs · 149 KB | 307 µs · 150 KB | 324 µs · 147 KB |
| Help | 343 µs · 118 KB | 367 µs · 130 KB | 437 µs · 230 KB | 957 µs · 1.2 MB |
| Settings | 87 µs · 24 KB | 100 µs · 24 KB | 88.5 µs · 24 KB | 88.3 µs · 24 KB |
| Brag view | 300 µs · 78 KB | 288 µs · 78 KB | 289 µs · 79 KB | 321 µs · 78 KB |
| Review details | 376 µs · 140 KB | 380 µs · 164 KB | 500 µs · 362 KB | 1.52 ms · 2.4 MB |
| Link menu | 1.03 ms · 266 KB | 1.05 ms · 278 KB | 1.54 ms · 390 KB | 29.3 ms · 6.5 MB |
| Delete confirm | 26.1 µs · 8 KB | 26.5 µs · 8 KB | 22.7 µs · 8 KB | 25.2 µs · 8 KB |
| Error | 330 µs · 77 KB | 360 µs · 76 KB | 414 µs · 87 KB | 826 µs · 208 KB |

### Typing in a long note

One keystroke plus redraw.

| Note length | Cursor at top | Cursor at bottom |
|---|---|---|
| 500 lines | 1.68 ms · 815 KB | 4.79 ms · 2.0 MB |
| 2000 lines | 5.36 ms · 2.7 MB | 18.4 ms · 7.6 MB |
| 10000 lines | 790 ms · 81.3 MB | 101 ms · 37.2 MB |

## History

One column per run, newest first; re-recording a version replaces its column. Rows are every scenario at 1k and at 10k notes; the 10k table adds process start, peak memory and typing in a long note.

<!-- history:start -->
### 1k notes

| Scenario | v0.0.153<br>2026-10-11<br>`921e69c`<br>Apple M1 (Virtual) | v0.0.152<br>2026-10-10<br>`c562e46`<br>Apple M1 (Virtual) |
|---|---|---|
| Startup | 22.5 ms · 9.4 MB | 22.7 ms · 9.5 MB |
| Startup heap | 1.3 MB | 1.3 MB |
| Load notes | 38.5 ms · 12.0 MB | 32.3 ms · 12.0 MB |
| Load reviews | 38.3 ms · 5.4 MB | 36.6 ms · 5.3 MB |
| Navigate j/k | 1.57 ms · 641 KB | 1.35 ms · 654 KB |
| Pulse tick | 403 µs · 106 KB | 237 µs · 99 KB |
| Header | 69.8 µs · 12 KB | 64.9 µs · 13 KB |
| Preview open | 1.35 ms · 704 KB | 999 µs · 744 KB |
| Preview next | 1.74 ms · 1.1 MB | 1.4 ms · 1.2 MB |
| Quick actions open | 1.81 ms · 533 KB | 1.54 ms · 527 KB |
| Quick actions move | 1.75 ms · 387 KB | 1.14 ms · 369 KB |
| Search keystroke | 1.4 ms · 479 KB | 929 µs · 468 KB |
| Save note | 2.44 ms · 509 KB | 1.93 ms · 494 KB |
| Delete note | 1.47 ms · 393 KB | 1.19 ms · 380 KB |
| Archive | 307 µs · 150 KB | 121 µs · 131 KB |
| Help | 437 µs · 230 KB | 307 µs · 250 KB |
| Settings | 88.5 µs · 24 KB | 62.7 µs · 24 KB |
| Brag view | 289 µs · 79 KB | 116 µs · 79 KB |
| Review details | 500 µs · 362 KB | 296 µs · 398 KB |
| Link menu | 1.54 ms · 390 KB | 967 µs · 371 KB |
| Delete confirm | 22.7 µs · 8 KB | 14.6 µs · 8 KB |
| Error | 414 µs · 87 KB | 203 µs · 86 KB |

### 10k notes

| Scenario | v0.0.153<br>2026-10-11<br>`921e69c`<br>Apple M1 (Virtual) | v0.0.152<br>2026-10-10<br>`c562e46`<br>Apple M1 (Virtual) |
|---|---|---|
| Process start | 4.5 ms | 3.99 ms |
| Peak memory | 7.5 MB | 7.3 MB |
| Startup | 239 ms · 82.7 MB | 215 ms · 82.8 MB |
| Startup heap | 6.4 MB | 6.4 MB |
| Load notes | 735 ms · 119.9 MB | 362 ms · 119.8 MB |
| Load reviews | 39 ms · 5.4 MB | 36.8 ms · 5.3 MB |
| Navigate j/k | 30 ms · 8.7 MB | 23.5 ms · 8.6 MB |
| Pulse tick | 922 µs · 222 KB | 572 µs · 208 KB |
| Header | 344 µs · 12 KB | 260 µs · 13 KB |
| Preview open | 17.2 ms · 6.5 MB | 16.6 ms · 6.6 MB |
| Preview next | 5.36 ms · 6.5 MB | 3.79 ms · 6.5 MB |
| Quick actions open | 31.6 ms · 7.8 MB | 28.3 ms · 7.5 MB |
| Quick actions move | 32.1 ms · 6.5 MB | 32.4 ms · 6.3 MB |
| Search keystroke | 3.71 ms · 1.0 MB | 2.76 ms · 1017 KB |
| Save note | 28.8 ms · 6.9 MB | 23.2 ms · 6.7 MB |
| Delete note | 27.2 ms · 6.7 MB | 24.9 ms · 6.7 MB |
| Archive | 324 µs · 147 KB | 157 µs · 129 KB |
| Help | 957 µs · 1.2 MB | 743 µs · 1.2 MB |
| Settings | 88.3 µs · 24 KB | 61.7 µs · 24 KB |
| Brag view | 321 µs · 78 KB | 108 µs · 77 KB |
| Review details | 1.52 ms · 2.4 MB | 1.22 ms · 2.4 MB |
| Link menu | 29.3 ms · 6.5 MB | 19.3 ms · 6.2 MB |
| Delete confirm | 25.2 µs · 8 KB | 14.1 µs · 8 KB |
| Error | 826 µs · 208 KB | 497 µs · 191 KB |
| Typing 500 lines, cursor top | 1.68 ms · 815 KB | 1.62 ms · 799 KB |
| Typing 500 lines, cursor bottom | 4.79 ms · 2.0 MB | 4.85 ms · 2.0 MB |
| Typing 2000 lines, cursor top | 5.36 ms · 2.7 MB | 5.23 ms · 2.6 MB |
| Typing 2000 lines, cursor bottom | 18.4 ms · 7.6 MB | 18.1 ms · 7.6 MB |
| Typing 10000 lines, cursor top | 790 ms · 81.3 MB | 984 ms · 81.2 MB |
| Typing 10000 lines, cursor bottom | 101 ms · 37.2 MB | 101 ms · 37.2 MB |

<!-- history:end -->
