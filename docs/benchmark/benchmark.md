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

v0.0.158 `dfe7a9d` · 2026-10-14 19:53 UTC · Apple M1 (Virtual)

### Process

| Process start (`digest --help`) | Peak memory |
|---|---|
| 7.23 ms | 7.5 MB |

### Startup and loading

Columns are notes on disk (reviews for Load reviews, at most 1000).

| Scenario | 1 | 100 | 1000 | 10000 |
|---|---|---|---|---|
| Startup | 1.2 ms · 793 KB | 6.23 ms · 2.0 MB | 43.5 ms · 9.8 MB | 456 ms · 85.6 MB |
| Startup heap | 0.6 MB | 0.7 MB | 1.3 MB | 6.6 MB |
| Load notes | 118 µs · 17 KB | 5.62 ms · 1.2 MB | 53.8 ms · 12.0 MB | 567 ms · 120.1 MB |
| Load reviews | 42.3 µs · 6 KB | 4.9 ms · 529 KB | 60.4 ms · 5.4 MB | 54.4 ms · 5.4 MB |

### Redraw and navigation

Columns are notes on disk (reviews for Load reviews, at most 1000).

| Scenario | 1 | 100 | 1000 | 10000 |
|---|---|---|---|---|
| Navigate j/k | 1.53 ms · 319 KB | 1.77 ms · 390 KB | 4.52 ms · 1.0 MB | 68.6 ms · 13.5 MB |
| Pulse tick | 481 µs · 95 KB | 516 µs · 95 KB | 540 µs · 107 KB | 1.95 ms · 276 KB |
| Header | 53.8 µs · 11 KB | 70.8 µs · 12 KB | 93.4 µs · 12 KB | 404 µs · 12 KB |
| Preview open | 1.03 ms · 309 KB | 1.29 ms · 403 KB | 5.01 ms · 1.3 MB | 71.1 ms · 13.9 MB |
| Preview next | 844 µs · 338 KB | 2 ms · 732 KB | 5.35 ms · 2.2 MB | 50.4 ms · 17.5 MB |

### Actions

Columns are notes on disk (reviews for Load reviews, at most 1000).

| Scenario | 1 | 100 | 1000 | 10000 |
|---|---|---|---|---|
| Quick actions open | 1.38 ms · 295 KB | 1.39 ms · 342 KB | 3.6 ms · 800 KB | 54.5 ms · 11.0 MB |
| Quick actions move | 1.1 ms · 264 KB | 1.41 ms · 285 KB | 2.24 ms · 481 KB | 37.2 ms · 7.5 MB |
| Search keystroke | 1.26 ms · 370 KB | 1.69 ms · 422 KB | 1.94 ms · 485 KB | 5.56 ms · 1.0 MB |
| Save note | 1.53 ms · 252 KB | 2.01 ms · 291 KB | 3.67 ms · 609 KB | 52.6 ms · 8.0 MB |
| Delete note | 1.46 ms · 216 KB | 1.34 ms · 244 KB | 2.22 ms · 486 KB | 41.6 ms · 7.8 MB |

### Screens and overlays

Columns are notes on disk (reviews for Load reviews, at most 1000).

| Scenario | 1 | 100 | 1000 | 10000 |
|---|---|---|---|---|
| Archive | 394 µs · 148 KB | 491 µs · 148 KB | 457 µs · 148 KB | 446 µs · 146 KB |
| Help | 540 µs · 127 KB | 668 µs · 153 KB | 1.57 ms · 417 KB | 11.4 ms · 3.0 MB |
| Settings | 142 µs · 27 KB | 138 µs · 27 KB | 136 µs · 28 KB | 134 µs · 27 KB |
| Brag view | 378 µs · 79 KB | 473 µs · 78 KB | 447 µs · 79 KB | 392 µs · 78 KB |
| Review details | 541 µs · 152 KB | 632 µs · 218 KB | 2.59 ms · 862 KB | 24.9 ms · 7.2 MB |
| Link menu | 1.32 ms · 266 KB | 1.33 ms · 288 KB | 2.45 ms · 483 KB | 46.8 ms · 7.6 MB |
| Delete confirm | 33.6 µs · 8 KB | 34.6 µs · 8 KB | 33.7 µs · 8 KB | 33.3 µs · 8 KB |
| Error | 521 µs · 77 KB | 603 µs · 77 KB | 628 µs · 88 KB | 1.31 ms · 241 KB |

### Typing in a long note

One keystroke plus redraw.

| Note length | Cursor at top | Cursor at bottom |
|---|---|---|
| 500 lines | 1.45 ms · 268 KB | 1.35 ms · 267 KB |
| 2000 lines | 1.55 ms · 478 KB | 1.43 ms · 475 KB |
| 10000 lines | 2.52 ms · 1.6 MB | 2.95 ms · 1.6 MB |

## History

One column per run, newest first; re-recording a version replaces its column. Rows are every scenario at 1k and at 10k notes; the 10k table adds process start, peak memory and typing in a long note.

<!-- history:start -->
### 1k notes

| Scenario | v0.0.158<br>2026-10-14<br>`dfe7a9d`<br>Apple M1 (Virtual) | v0.0.157<br>2026-10-12<br>`588dd34`<br>Apple M2 Pro (Virtual) | v0.0.156<br>2026-10-11<br>`a613175`<br>Apple M1 (Virtual) | v0.0.155<br>2026-10-11<br>`578beec`<br>Apple M1 (Virtual) | v0.0.154<br>2026-10-11<br>`d8d8840`<br>Apple M1 (Virtual) | v0.0.153<br>2026-10-11<br>`921e69c`<br>Apple M1 (Virtual) | v0.0.152<br>2026-10-10<br>`c562e46`<br>Apple M1 (Virtual) |
|---|---|---|---|---|---|---|---|
| Startup | 43.5 ms · 9.8 MB | 19 ms · 9.8 MB | 41.2 ms · 9.5 MB | 29.6 ms · 9.4 MB | 29.2 ms · 9.4 MB | 22.5 ms · 9.4 MB | 22.7 ms · 9.5 MB |
| Startup heap | 1.3 MB | 1.3 MB | 1.3 MB | 1.3 MB | 1.3 MB | 1.3 MB | 1.3 MB |
| Load notes | 53.8 ms · 12.0 MB | 23.3 ms · 12.1 MB | 54.3 ms · 12.0 MB | 26.1 ms · 12.0 MB | 42.2 ms · 12.0 MB | 38.5 ms · 12.0 MB | 32.3 ms · 12.0 MB |
| Load reviews | 60.4 ms · 5.4 MB | 29.4 ms · 5.4 MB | 62.2 ms · 5.4 MB | 34.9 ms · 5.4 MB | 36.9 ms · 5.4 MB | 38.3 ms · 5.4 MB | 36.6 ms · 5.3 MB |
| Navigate j/k | 4.52 ms · 1.0 MB | 3.58 ms · 1.0 MB | 1.9 ms · 643 KB | 1.89 ms · 638 KB | 1.65 ms · 641 KB | 1.57 ms · 641 KB | 1.35 ms · 654 KB |
| Pulse tick | 540 µs · 107 KB | 507 µs · 106 KB | 501 µs · 106 KB | 402 µs · 106 KB | 416 µs · 106 KB | 403 µs · 106 KB | 237 µs · 99 KB |
| Header | 93.4 µs · 12 KB | 76 µs · 12 KB | 86.2 µs · 12 KB | 75.7 µs · 12 KB | 79 µs · 12 KB | 69.8 µs · 12 KB | 64.9 µs · 13 KB |
| Preview open | 5.01 ms · 1.3 MB | 3.88 ms · 1.3 MB | 1.48 ms · 717 KB | 1.16 ms · 706 KB | 1.14 ms · 715 KB | 1.35 ms · 704 KB | 999 µs · 744 KB |
| Preview next | 5.35 ms · 2.2 MB | 6.6 ms · 2.2 MB | 2.02 ms · 1.1 MB | 1.56 ms · 1.1 MB | 1.6 ms · 1.1 MB | 1.74 ms · 1.1 MB | 1.4 ms · 1.2 MB |
| Quick actions open | 3.6 ms · 800 KB | 2.89 ms · 792 KB | 1.99 ms · 533 KB | 1.45 ms · 528 KB | 1.52 ms · 531 KB | 1.81 ms · 533 KB | 1.54 ms · 527 KB |
| Quick actions move | 2.24 ms · 481 KB | 1.82 ms · 472 KB | 1.84 ms · 389 KB | 1.36 ms · 387 KB | 1.43 ms · 387 KB | 1.75 ms · 387 KB | 1.14 ms · 369 KB |
| Search keystroke | 1.94 ms · 485 KB | 1.42 ms · 492 KB | 2.62 ms · 484 KB | 1.51 ms · 485 KB | 1.41 ms · 484 KB | 1.4 ms · 479 KB | 929 µs · 468 KB |
| Save note | 3.67 ms · 609 KB | 3.54 ms · 599 KB | 2.79 ms · 511 KB | 2.16 ms · 509 KB | 2.56 ms · 510 KB | 2.44 ms · 509 KB | 1.93 ms · 494 KB |
| Delete note | 2.22 ms · 486 KB | 1.85 ms · 477 KB | 1.87 ms · 394 KB | 1.56 ms · 393 KB | 1.36 ms · 393 KB | 1.47 ms · 393 KB | 1.19 ms · 380 KB |
| Archive | 457 µs · 148 KB | 310 µs · 154 KB | 358 µs · 150 KB | 309 µs · 150 KB | 380 µs · 149 KB | 307 µs · 150 KB | 121 µs · 131 KB |
| Help | 1.57 ms · 417 KB | 1.02 ms · 422 KB | 647 µs · 230 KB | 431 µs · 233 KB | 484 µs · 231 KB | 437 µs · 230 KB | 307 µs · 250 KB |
| Settings | 136 µs · 28 KB | 86.7 µs · 25 KB | 115 µs · 24 KB | 92.5 µs · 24 KB | 79.8 µs · 24 KB | 88.5 µs · 24 KB | 62.7 µs · 24 KB |
| Brag view | 447 µs · 79 KB | 293 µs · 80 KB | 333 µs · 79 KB | 298 µs · 79 KB | 277 µs · 79 KB | 289 µs · 79 KB | 116 µs · 79 KB |
| Review details | 2.59 ms · 862 KB | 2.25 ms · 853 KB | 594 µs · 361 KB | 443 µs · 361 KB | 449 µs · 362 KB | 500 µs · 362 KB | 296 µs · 398 KB |
| Link menu | 2.45 ms · 483 KB | 1.72 ms · 474 KB | 2 ms · 391 KB | 2.24 ms · 390 KB | 1.28 ms · 389 KB | 1.54 ms · 390 KB | 967 µs · 371 KB |
| Delete confirm | 33.7 µs · 8 KB | 25.7 µs · 8 KB | 26.4 µs · 8 KB | 22.1 µs · 8 KB | 21.7 µs · 8 KB | 22.7 µs · 8 KB | 14.6 µs · 8 KB |
| Error | 628 µs · 88 KB | 421 µs · 87 KB | 587 µs · 87 KB | 350 µs · 87 KB | 357 µs · 87 KB | 414 µs · 87 KB | 203 µs · 86 KB |

### 10k notes

| Scenario | v0.0.158<br>2026-10-14<br>`dfe7a9d`<br>Apple M1 (Virtual) | v0.0.157<br>2026-10-12<br>`588dd34`<br>Apple M2 Pro (Virtual) | v0.0.156<br>2026-10-11<br>`a613175`<br>Apple M1 (Virtual) | v0.0.155<br>2026-10-11<br>`578beec`<br>Apple M1 (Virtual) | v0.0.154<br>2026-10-11<br>`d8d8840`<br>Apple M1 (Virtual) | v0.0.153<br>2026-10-11<br>`921e69c`<br>Apple M1 (Virtual) | v0.0.152<br>2026-10-10<br>`c562e46`<br>Apple M1 (Virtual) |
|---|---|---|---|---|---|---|---|
| Process start | 7.23 ms | 7.58 ms | 5.49 ms | 3.59 ms | 3.55 ms | 4.5 ms | 3.99 ms |
| Peak memory | 7.5 MB | 7.6 MB | 7.4 MB | 7.4 MB | 7.3 MB | 7.5 MB | 7.3 MB |
| Startup | 456 ms · 85.6 MB | 199 ms · 85.7 MB | 367 ms · 82.8 MB | 202 ms · 82.7 MB | 340 ms · 82.8 MB | 239 ms · 82.7 MB | 215 ms · 82.8 MB |
| Startup heap | 6.6 MB | 6.6 MB | 6.4 MB | 6.4 MB | 6.4 MB | 6.4 MB | 6.4 MB |
| Load notes | 567 ms · 120.1 MB | 275 ms · 120.4 MB | 534 ms · 119.5 MB | 298 ms · 119.8 MB | 435 ms · 119.6 MB | 735 ms · 119.9 MB | 362 ms · 119.8 MB |
| Load reviews | 54.4 ms · 5.4 MB | 29.2 ms · 5.4 MB | 68.9 ms · 5.4 MB | 42.4 ms · 5.4 MB | 37.4 ms · 5.4 MB | 39 ms · 5.4 MB | 36.8 ms · 5.3 MB |
| Navigate j/k | 68.6 ms · 13.5 MB | 47.9 ms · 13.3 MB | 32.4 ms · 8.7 MB | 29.7 ms · 8.7 MB | 29.4 ms · 8.8 MB | 30 ms · 8.7 MB | 23.5 ms · 8.6 MB |
| Pulse tick | 1.95 ms · 276 KB | 941 µs · 227 KB | 1.02 ms · 237 KB | 768 µs · 220 KB | 985 µs · 228 KB | 922 µs · 222 KB | 572 µs · 208 KB |
| Header | 404 µs · 12 KB | 282 µs · 12 KB | 331 µs · 12 KB | 251 µs · 12 KB | 270 µs · 12 KB | 344 µs · 12 KB | 260 µs · 13 KB |
| Preview open | 71.1 ms · 13.9 MB | 40.1 ms · 14.3 MB | 20.9 ms · 6.5 MB | 19.9 ms · 6.5 MB | 16.3 ms · 6.5 MB | 17.2 ms · 6.5 MB | 16.6 ms · 6.6 MB |
| Preview next | 50.4 ms · 17.5 MB | 40.7 ms · 17.5 MB | 4.45 ms · 6.5 MB | 4 ms · 6.5 MB | 3.91 ms · 6.5 MB | 5.36 ms · 6.5 MB | 3.79 ms · 6.5 MB |
| Quick actions open | 54.5 ms · 11.0 MB | 42.7 ms · 11.0 MB | 58.5 ms · 8.1 MB | 35.1 ms · 7.8 MB | 32.6 ms · 7.8 MB | 31.6 ms · 7.8 MB | 28.3 ms · 7.5 MB |
| Quick actions move | 37.2 ms · 7.5 MB | 34.8 ms · 7.4 MB | 48.7 ms · 6.6 MB | 32.2 ms · 6.5 MB | 29.3 ms · 6.4 MB | 32.1 ms · 6.5 MB | 32.4 ms · 6.3 MB |
| Search keystroke | 5.56 ms · 1.0 MB | 3.42 ms · 1.0 MB | 5.31 ms · 1.0 MB | 3.25 ms · 1.0 MB | 4.56 ms · 1.0 MB | 3.71 ms · 1.0 MB | 2.76 ms · 1017 KB |
| Save note | 52.6 ms · 8.0 MB | 34.5 ms · 7.8 MB | 43.1 ms · 7.1 MB | 28.6 ms · 6.9 MB | 27 ms · 6.9 MB | 28.8 ms · 6.9 MB | 23.2 ms · 6.7 MB |
| Delete note | 41.6 ms · 7.8 MB | 33.5 ms · 7.7 MB | 66.8 ms · 6.8 MB | 28.8 ms · 6.8 MB | 26.2 ms · 6.7 MB | 27.2 ms · 6.7 MB | 24.9 ms · 6.7 MB |
| Archive | 446 µs · 146 KB | 284 µs · 148 KB | 355 µs · 147 KB | 325 µs · 147 KB | 374 µs · 147 KB | 324 µs · 147 KB | 157 µs · 129 KB |
| Help | 11.4 ms · 3.0 MB | 6.24 ms · 3.0 MB | 958 µs · 1.2 MB | 935 µs · 1.2 MB | 949 µs · 1.2 MB | 957 µs · 1.2 MB | 743 µs · 1.2 MB |
| Settings | 134 µs · 27 KB | 79.1 µs · 24 KB | 107 µs · 24 KB | 96.6 µs · 24 KB | 75.9 µs · 24 KB | 88.3 µs · 24 KB | 61.7 µs · 24 KB |
| Brag view | 392 µs · 78 KB | 252 µs · 78 KB | 402 µs · 78 KB | 272 µs · 77 KB | 324 µs · 77 KB | 321 µs · 78 KB | 108 µs · 77 KB |
| Review details | 24.9 ms · 7.2 MB | 17.5 ms · 7.2 MB | 2.87 ms · 2.4 MB | 1.57 ms · 2.4 MB | 1.68 ms · 2.4 MB | 1.52 ms · 2.4 MB | 1.22 ms · 2.4 MB |
| Link menu | 46.8 ms · 7.6 MB | 34.7 ms · 7.3 MB | 34.3 ms · 6.6 MB | 33.2 ms · 6.5 MB | 34.3 ms · 6.5 MB | 29.3 ms · 6.5 MB | 19.3 ms · 6.2 MB |
| Delete confirm | 33.3 µs · 8 KB | 22.2 µs · 8 KB | 27 µs · 8 KB | 20.1 µs · 8 KB | 21.6 µs · 8 KB | 25.2 µs · 8 KB | 14.1 µs · 8 KB |
| Error | 1.31 ms · 241 KB | 1.08 ms · 220 KB | 2.29 ms · 254 KB | 795 µs · 201 KB | 679 µs · 199 KB | 826 µs · 208 KB | 497 µs · 191 KB |
| Typing 500 lines, cursor top | 1.45 ms · 268 KB | 1.18 ms · 291 KB | 1.25 ms · 268 KB | 2.4 ms · 812 KB | 1.64 ms · 814 KB | 1.68 ms · 815 KB | 1.62 ms · 799 KB |
| Typing 500 lines, cursor bottom | 1.35 ms · 267 KB | 1.17 ms · 282 KB | 1.18 ms · 270 KB | 6.54 ms · 2.1 MB | 4.97 ms · 2.1 MB | 4.79 ms · 2.0 MB | 4.85 ms · 2.0 MB |
| Typing 2000 lines, cursor top | 1.55 ms · 478 KB | 1.47 ms · 487 KB | 1.57 ms · 480 KB | 5.99 ms · 2.7 MB | 5.59 ms · 2.7 MB | 5.36 ms · 2.7 MB | 5.23 ms · 2.6 MB |
| Typing 2000 lines, cursor bottom | 1.43 ms · 475 KB | 1.15 ms · 489 KB | 1.38 ms · 477 KB | 21.5 ms · 7.6 MB | 18.5 ms · 7.6 MB | 18.4 ms · 7.6 MB | 18.1 ms · 7.6 MB |
| Typing 10000 lines, cursor top | 2.52 ms · 1.6 MB | 2.06 ms · 1.6 MB | 2.61 ms · 1.6 MB | 934 ms · 81.3 MB | 846 ms · 81.3 MB | 790 ms · 81.3 MB | 984 ms · 81.2 MB |
| Typing 10000 lines, cursor bottom | 2.95 ms · 1.6 MB | 2.63 ms · 1.6 MB | 3.01 ms · 1.6 MB | 93.4 ms · 37.2 MB | 89.2 ms · 37.2 MB | 101 ms · 37.2 MB | 101 ms · 37.2 MB |

<!-- history:end -->
