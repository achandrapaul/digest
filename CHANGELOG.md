# Changelog

## v0.0.159 — 2026-10-25

### Changes
- [50facf8](https://github.com/achandrapaul/digest/commit/50facf8699bbd945b0952d9140e73feed6fb7f7b) fix: digest bootstrap stops at a config stage on load errors; config and doctor stages are not retried
- [1d2354a](https://github.com/achandrapaul/digest/commit/1d2354a4b050b2ac5648be0b65d8875132bdb280) adding summarise
- [d5581dc](https://github.com/achandrapaul/digest/commit/d5581dc16baaa3038bb385417b8b2c9a373add25) feat: digest summary --exclude=pr leaves out #my-pr and #pr-review notes; unknown --exclude values fail with exit 1
- [ba8e9d3](https://github.com/achandrapaul/digest/commit/ba8e9d3021a89cdb201b8984c974184fdd113d95) feat: add digest summary command printing the previous work day's done notes, pending notes and today's active and closed notes, with --json, sharing the dashboard's note grouping

## v0.0.158 — 2026-10-14

### Changes
- [5060fe2](https://github.com/achandrapaul/digest/commit/5060fe29673c414b1ccd5a8cc756d51c09067f4b) feat: PR review notifications run digest focus <org>:<repo>:<title>, which raises the running digest's tmux pane or terminal and selects that PR once no edit is open; .app.state.json and the focus file are written through a queued lock; the pre-commit hook lints before testing; tests use neutral example repo names
- [25d8e9e](https://github.com/achandrapaul/digest/commit/25d8e9e0c71d395f9a5f4608298b4dd6b38cdc31) fix: dry-run exit file is written atomically so a signalled command reports 143, Raycast missing error starts lowercase
- [e185f47](https://github.com/achandrapaul/digest/commit/e185f4733dc4b9f405ebbe8427d442c993b8c6e6) feat: h groups dashboard notes under tag headers, saved in .app.state.json; My PR notes are summarised Own:<state>:<org>:<repo>:<title>
  - h (hidden) toggles a #tag header over each run of notes in Yesterday, Pending, Active items for today and Closed Today; sorting is unchanged, and Active items for today gathers each tag together. The choice is saved as group_by_tag in .app.state.json.
  - My PR note summaries read Own:DRAFT|OPEN:<org>:<repo>:<title> while open and Own:MERGED|CLOSED:… once the PR ends; the note id and the rest of the body stay the same, and the seen-PR rebuild reads titles from both the new and the old NewPR: summaries.
- [91a0bb8](https://github.com/achandrapaul/digest/commit/91a0bb8d82897c1d895b07f867ce4aa7a2b75ea3) fix: PR review installs deps through mise when nvm is missing and drops invalid MCP allow rules; Closed Today groups by tag
  - With an .nvmrc, the review's dependency install uses nvm only when $NVM_DIR/nvm.sh exists, otherwise mise exec node@<version>, otherwise the node on PATH; MISE_* settings reach the install.
  - Plugin allow rules name each MCP server from the plugin's .mcp.json or plugin.json (mcp__plugin_<plugin>_<server>__*), so Claude no longer ignores mcp__plugin_<plugin>_*; plugins without servers get only their Skill rule.
  - Closed Today lists #manual first, then #my-pr, #pr-review and other tags, newest done first within each; marking a note done records the current time on the viewed day.
- [bd74a2c](https://github.com/achandrapaul/digest/commit/bd74a2cf4fd8555711e7b9a38dcab2fcbd64ebeb) feat: digest install spotlight adds an Add Digest Note Apple Shortcut for Spotlight and the menu bar
  - digest install spotlight writes, signs and opens an "Add Digest Note" shortcut that asks for the note text and runs digest note add; it is pinned to the Shortcuts menu bar menu, takes one Add Shortcut click and needs Allow Running Scripts in Shortcuts → Settings → Advanced.
  - Without the shortcuts command it exits 1; digest uninstall spotlight removes its files and says to delete the shortcut in Shortcuts. It works without Raycast and is not part of bootstrap.
  - doctor shows an optional spotlight row: missing, "shortcut not added; run: digest install spotlight", or the added shortcut.
- [2e5577f](https://github.com/achandrapaul/digest/commit/2e5577f6ca9fbc0cc04d0e0876168251b5cfbe3a) feat: f notices, confirms and help read start today / mark pending
  - f now reports "started today" / "marked pending" ("started N notes today" / "marked N notes pending" on a selection), and the selection confirm asks "Start this note today?" / "Mark this note pending?".
  - The shortcuts help line for f reads "earlier note: start today, or mark pending".
- [a609064](https://github.com/achandrapaul/digest/commit/a609064b81d4c8dea1ccb684202edaa46cd5d474) feat: digest note add from the CLI, digest install raycast for an Add Digest Note script command, f hint reads start today / mark pending
  - digest note add <text> saves a manual active note for today the same way the dashboard's a does: first line is the summary, the rest the body; empty text fails and a locked note only warns.
  - digest install raycast writes an "Add Digest Note" script command to ~/.config/digest/raycast that runs digest note add, and exits 1 when Raycast is missing; digest uninstall raycast removes it, and doctor shows an optional raycast row that never fails.
  - The f row hint on an earlier note now reads "start today", and "mark pending" once it's started today.
- [c3af328](https://github.com/achandrapaul/digest/commit/c3af32827b86321b8bc3e4ab6a37f143b9086d9d) feat: shift+J/K multi-select notes on the dashboard, with confirmed bulk done, f and delete
  - shift+J/K or shift+down/up selects an anchored range of notes across Yesterday, Carried, Active and Closed; j/k, paging, p/n/t and esc cancel it.
  - space marks the active notes in the range done, or all active when every note is done; f works on eligible notes today or moves them back to pending; d archives them. With a selection each asks first and clears the selection on confirm; single rows keep their existing behaviour.
  - ctrl+y copies the selected notes' summary and body and keeps the selection; hint shows (shift)multiple selection.

## v0.0.157 — 2026-10-12

### Changes
- [369a37d](https://github.com/achandrapaul/digest/commit/369a37dab1f0d48ec904da9b998b5a3bc0c10dcf) feat: secret redaction before AI sends, f to pull pending notes into today, preview headers show row tags
  - Notes and brag facts sent to Claude are redacted (built-in patterns plus redact_patterns); "Added Today" is now "Active items for today" and f toggles started_working_on, with done recording the day and the preview listing worked-on dates under its header.
  - Preview headers carry their row's tags (no #general fallback), the dashboard ctrl+c warning is a single centred line, and Pending, Carried Over sorts by source tag then newest created, both descending.

## v0.0.156 — 2026-10-11

### Changes
- [310f233](https://github.com/achandrapaul/digest/commit/310f2338015251a84dc0d4050b6ab41adff4515d) perf: editor keeps per-row wrapped heights instead of a wrap cache, typing in a 10k-line note drops from 100-940 ms to about 2 ms

## v0.0.155 — 2026-10-11

### Changes
- [9f36007](https://github.com/achandrapaul/digest/commit/9f360078a105a538a42bafe327fe494bd950c24a) chore: release:trigger branches off main, bumps and opens the release PR

## v0.0.154 — 2026-10-11

### Changes
- [dfbf9b6](https://github.com/achandrapaul/digest/commit/dfbf9b6c5d8debcd12964a8a608ef304c21d468e) ci: release is safe to rerun and a slow go proxy no longer blocks the tap
- [6fe6527](https://github.com/achandrapaul/digest/commit/6fe6527d86dfaf3b647d02db3db759ec9b5251d7) ci: release workflow runs on v* tag pushes and accepts tags without the v prefix

## v0.0.153 — 2026-10-11

### Changes
- [2db9262](https://github.com/achandrapaul/digest/commit/2db926298f76ec797e9e9b23bd0a8537c856170f) test: compare first_note_created by instant in migrate test
- [558c318](https://github.com/achandrapaul/digest/commit/558c318883e5074cf603159e837a0fd77402bc43) chore: pre-commit hook runs the tests, mise run hooks installs it
- [b2299bc](https://github.com/achandrapaul/digest/commit/b2299bc9b9a3e2cb97c104cbf0f6b9830112ca86) refactor: shared system exec, process, clipboard and tail helpers, jobs runtime moved into pkg/jobs, list cursor, config template helpers and PR source interface; first_note_created set once from the first manual note, TUI requires app state and advises migrate or bootstrap, bootstrap writes an empty state, migrate fills only a null date; track cmd/digest sources
- [763971c](https://github.com/achandrapaul/digest/commit/763971c0221047360d3d12aac41fc278f067f937) feat: digest bootstrap, tag-based release notes, git hooks and fsmonitor off for job git calls, unknown config key warnings, migrate carries reminders and automation state and adds google calendar once, TUI split into per-area state with a mode stack, CI race, lint and mod checks, go 1.27.1; remove git auto-sync
- [751594c](https://github.com/achandrapaul/digest/commit/751594ceb771773d02d8c5c4e37147088abd264b) fix: tap formula uses the achandrapaul module path

## v0.0.152 — 2026-10-10

### Changes
- [afcae37](https://github.com/achandrapaul/digest/commit/afcae37f7d8ed00fa4f48afad5199e12ef146e7b) fix: streak ignores the zone notes were saved in
  - habit.Streak keys finished days by the day in the current zone, so notes read back as UTC or saved under another offset still count
  - the migrate dry-run test compares first_note_created by instant instead of by location
- [fa9068c](https://github.com/achandrapaul/digest/commit/fa9068c1ff1f3a41dd27ab170b0a50043965a1fb) feat: achandrapaul module path, per-job options, tag-then-publish releases, on-demand note loading, glamour-free 7 MB binary
  Module and release
  - module path is now github.com/achandrapaul/digest; Homebrew tap, go install, release links and ldflags follow
  - a push to main that changes .version runs the new build workflow: vet, test and tag v<version>, nothing published
  - the release workflow is now manual (workflow_dispatch, optional tag, latest tag by default): checks out the tag, publishes binaries, warms the Go module proxy, updates the tap, then records the benchmark
  - release:commit bumps .version (patch by default), writes the CHANGELOG.md section, commits as release:<level>, pushes, and opens a PR when off main
  - release notes <version> writes its CHANGELOG.md section once and prints it; the changelog subcommand is gone
  - the last release is the newest published GitHub release tag, falling back to the last .version commit
  - the changelog is one Changes list of linked short hashes with commit bodies; non-conventional commits are kept, release, changelog and benchmark commits are left out
  Binary size
  - glamour replaced by a small lipgloss markdown renderer for headings, bold, italic, code, links, lists, tasks, quotes, fences and rules, wrapped to the panel width
  - notification icon embedded as icon.png instead of drawn with image/png at runtime; a test fails when it drifts from the banner and -update regenerates it
  - chroma, goldmark, bluemonday and regexp2 dropped; the stripped binary goes from 14.0 MB to 7.1 MB
  CLI
  - global flags parsed strictly: bad flags print an error and usage and exit 1, -h, --help and digest help print usage and exit 0
  - global --dry-run, refused with a message by commands that can't dry-run
  - digest migrate --dry-run prints what it would do and changes nothing
  - janitor --no-quarantine (or no_quarantine: true) deletes matches instead of quarantining them
  - every subcommand rejects unknown flags
  - digest init [--force] writes the default config.yaml, refusing an existing one unless --force backs it up to config.yaml.bak; it replaces export config and refuses --dry-run
  - digest init --automations backs up config.yaml and replaces only its automations section with the defaults and their full prompts
  - startup loads config with defaults instead of creating the file; load warnings show in the TUI header
  Config and jobs
  - branch-reaper, janitor and repo-sync are built in with their commands and dry-run commands; config.yaml sets only their options, and a job left out runs with digest's defaults
  - command or dry-run-command on a built-in job is a load error; custom jobs keep their commands
  - jobs take an options: map passed to the command as DIGEST_OPTION_<NAME>, lists comma-joined, from the dashboard and the shell; built-in jobs ignore unknown options
  - a roots option per built-in job (~/Projects for branch-reaper and repo-sync, ~ for janitor), overridden by --root; janitor no longer defaults to the current directory
  - the default config is config.yaml embedded from config.yaml.go, without a jobs section
  - janitor grace_days and patterns replace top-level retention_days and janitor_patterns; branch-reaper takes grace_days (default 7)
  - paths.Expand returns errors for unset $VARs and supports ~user; config, store and jobs report them
  - janitor fails on a missing root, records failed purges, deletes idle review clones with unreadable meta
  - repo-sync reports unusable roots, syncs the rest, fails only when none work, detects repos without remotes and finds the default branch via ls-remote
  - branch-reaper ignores untracked files when checking for a dirty repo and checks upstream containment in dry runs
  - system.RemoveAll and LockTree keep going past failures and report a count
  Migrate
  - moves retention_days and janitor_patterns into the janitor job's options through the YAML tree, adding an options-only janitor entry when none exists
  - removes built-in job commands, turning their --root values into the roots option and renaming branch reaper and repo sync; custom jobs stay
  - adds automation_type to built-in automations that lack it and the google calendar automation when it is missing, keeping prompts and match lists
  - reports skipped notes instead of aborting, and corrects first_note_created in app state
  Automations
  - every automation has an automation_type: jira ticket, confluence confluence, google doc doc, google calendar event, pr review reviewed; a custom automation without one fails to load
  - every automation except pr review needs a draft_prompt and a create_prompt
  - the create result's kind must equal the automation_type, and the note section is headed by it
  - the automated tag shows only for manual, my-pr, pr-review and the configured automation types
  - built-in automations, their full prompts and match lists come from the embedded config.yaml; create is gone from the match lists
  - new google calendar automation through the claude.ai Google Calendar connector; its draft stops with NO TIME: when the note names no date or start time
  Store and notes
  - unreadable or unparseable notes are reported as SkippedNotesError while the rest load
  - clearer front matter errors; unknown or inbox status loads as active; only note files are listed
  - ListDashboard takes the viewed and previous work day; ListDoneSince loads finished notes on demand
  - LoadByID prefers the active copy then the newest finished one; deleting a missing file succeeds
  - the dashboard keeps only active notes and those finished on the viewed and previous work day; archive, search and brag load every note on demand and drop them on return
  - the previous day is the previous configured work day; app state rebuilds off the UI thread; closed this week counts notes beyond the dashboard
  Dashboard and keys
  - pgdown and pgup move a full page, ctrl+d and ctrl+u half a page, in the dashboard, archive and editors
  - git strip column switching with h, l and arrows removed
  - row keys bind only when there are rows; c binds only when daily commits are on
  - every yes or no confirm shares the same keys, n and esc cancel everywhere
  - ctrl+c counts presses within a 2s window and any other key resets it
  - n goes at most one day past today; ctrl+r reloads the viewed day
  - the too-small screen follows the header width
  - day headings stay letter-spaced with a middle dot on past days, and yesterday counts from today
  - a note with a reminder offers @notify:off then @notify; choosing @notify again rewrites the reminder with the new interval
  Editors and setup
  - esc asks Discard changes? when the note, brag or automation editor has edits
  - saving an empty note is blocked; an empty inline edit asks to delete the note
  - tab inserts indentation in the editors; ctrl+y copies in the automation and reject-comment editors
  - the reject comment submits straight away
  - setup moves between visited steps with left and right, asks before undoing a changed field, backs up once per session, finishes with one action that installs notifications, and shows errors as notices
  Search
  - search previews reuse the normal preview, so edit, delete and copy work there; enter opens a result
  - a blank query shows nothing; date filters match created or updated
  Errors and messages
  - popups, previews, search and editors show errors as a one-line notice that clears on the next key
  - the message log colours entries and opens an expanded, scrollable view on a second !
  - browser and clipboard failures are reported, ctrl+y confirms copies
  - unreadable history, brag files and PR seen state are reported; the PR seen state rebuilds from existing notes
  Jobs and runs
  - run, dry run, stale job, retry and delete-draft confirms in one place; dry run now asks first
  - running a job during its dry run asks to stop the dry run first
  - failed and needs-reauth automation runs stay listed with badges; finished runs can be dismissed or retried
  - d on a job only dry-runs when it has a dry-run-command; the x draft-again key moved to the actions menu
  - review stop writes exit 143; review.Dismiss clears finished reviews
  PRs and notifications
  - PR alerts include PR opened for your new PRs and fire on every sync for new or changed PRs, the first sync included
  - PR note lookup falls back to a one-time scan of notes on disk
  - the action menu order is saved in app state instead of action-usage.json
  - the reminder interval is validated and capped, and saves without blocking
  Tests
  - new coverage for keys, confirms, editors, modals, resize, search, setup, jobs, reviews, PR notes, on-demand loading, markdown rendering, the embedded icon, config options, built-in jobs and roots, automation types, init, migrate config and jobs, day headings, store integrity, janitor, repo-sync and branch-reaper rules
- [072f17d](https://github.com/achandrapaul/digest/commit/072f17d5942951226dff5142c4d836a396f46984) fix: central file access with locked digest files, app state for streak and first note, Claude review findings via stdout, job rows aligned right, jobs and background runs survive app restarts, migrate logs locking
  - fix: central file access with locked digest files, app state for streak and first note, Claude review findings via stdout, job rows aligned right, jobs and background runs survive app restarts, migrate logs locking
  - ci: release only on a pushed version bump; mise run release:commit bumps and commits .version, the release workflow writes and commits CHANGELOG.md from the commits since the previous release, mise tasks use colon names, release and benchmark docs moved to docs/, benchmark history cleared into docs/benchmark/benchmark.md
- [0f48dbb](https://github.com/achandrapaul/digest/commit/0f48dbbff98d1a49dbaab3f425bbd77e08a8db83) feat: per-section files, git below today, esc dismisses errors, ctrl+r reload, single-note saves, ID-only PR note lookups, benchmark history tables
- [b30d08f](https://github.com/achandrapaul/digest/commit/b30d08f32969f0161097c2a233257bed0bb2ebd7) feat: status in note file names, faster saves, digest migrate, no pulse dot
- [cff6519](https://github.com/achandrapaul/digest/commit/cff651901d25036c30962cc21c3b2a53a7c632cc) feat: adding benchmark and make textarea rendering 28x faster
- [1c67af8](https://github.com/achandrapaul/digest/commit/1c67af8ad9f41af88a2f99d1962776ef1027402e) fix: faster editor and dashboard, safer reviews, read-only dry runs
  - editor: no 99-line limit, cached view, solid cursor, hints only on the dashboard
  - dashboard: pulse dot drawn after caching; run logs shown as plain text
  - reviews: owner in clone folder names, terminal escapes stripped, no SSH agent or npm token for installs, clone timeout and process-group kill
  - notes: deleting a note removes its automation files; note ids can't escape the automations folder
  - dry runs: no renames, pid cleanup, prune, gc, index writes or config writes
  - jobs: d only when a dry run exists, digest dry runs must pass --dry-run, configured digest commands use the running binary
- [abbcf55](https://github.com/achandrapaul/digest/commit/abbcf556fafb0d8144e87f5c893d4ce49ff9e729) fix: janitor removes leftover partial review clones
- [71fa6f9](https://github.com/achandrapaul/digest/commit/71fa6f95689ee16bc0e720729854cd53c85cc17c) fix: show o nvim on PR rows only once the clone has finished
- [bc1d6ea](https://github.com/achandrapaul/digest/commit/bc1d6eae406979c60e3583342625b6fa18a7b650) fix: open a note's links with o, picking from a list when there are several
- [70c09d5](https://github.com/achandrapaul/digest/commit/70c09d543d51a65aed90b1dc292a8506c5674da3) fix: enter edits pr-review notes in the preview, o opens the PR
- [5e868cc](https://github.com/achandrapaul/digest/commit/5e868cc340c14c12832e247588b63414014b539f) feat: choose the release version from commits and publish a changelog
- [fd649cb](https://github.com/achandrapaul/digest/commit/fd649cbe318c24f28e9c9c876391ae079fc4923e) feat: show release versions without the dev suffix
- [21ba9e8](https://github.com/achandrapaul/digest/commit/21ba9e85be68ea9c7da3435344192393aef305a5) feat: show the streak beside the logo, above the version
- [a97d0e7](https://github.com/achandrapaul/digest/commit/a97d0e7926f1ca5abfbbcbca56ef8371a3c6b856) docs: add step-by-step setup to the readme
- [d7218de](https://github.com/achandrapaul/digest/commit/d7218decffb6ab8ee1b45348878168428f07ca53) ci: give bump and release their own concurrency groups (#1)
  Co-authored-by: Anudeep Chandra Paul <18657542+AnudeepChPaul@users.noreply.github.com>
- [759ba7d](https://github.com/achandrapaul/digest/commit/759ba7dead7941c414510b3b455a415a510bdf43) test: doctor check sets up its own plugin cache
- [94ae261](https://github.com/achandrapaul/digest/commit/94ae2618fb0fffcf320f4b16119b871b823969ec) ci: test, bump and release with mise on GitHub Actions
- [c74b3b3](https://github.com/achandrapaul/digest/commit/c74b3b377fbc7396153012840c277994bbc48a0a) chore: release from public tarballs
- [51b0e68](https://github.com/achandrapaul/digest/commit/51b0e688650a26094ec3b114f1b128b68cc71d88) chore: MIT licence, README, module github.com/AnudeepChPaul/digest
- [ae9d5ff](https://github.com/achandrapaul/digest/commit/ae9d5ffbbf32455edabe5f63ed9ef46036810b79) fix: annotate release tags
- [ceca107](https://github.com/achandrapaul/digest/commit/ceca107a9cf0508f7355df98977b323efedc2adc) feat: setup form, digest install, ctrl+c guard, brew release task
- [8a160e9](https://github.com/achandrapaul/digest/commit/8a160e9bf9564cb33cb5c62100849c9dc788dcc9) feat: tighter PR status icon gap, rec:Changes label
- [e6b223f](https://github.com/achandrapaul/digest/commit/e6b223fb191bca7afd5558f004781550c6451432) another feature set complete
- [561d819](https://github.com/achandrapaul/digest/commit/561d819f0b87e9df0e133aa4eae56682ac7622e0) feat: my PRs section, re-review changes, selection underline, uniform row colour
  - My PR(s) block below the day columns with files, age, approval and CI
  - Re-review when the reviewed head moved, with changes since the review
  - Board order yesterday, git, today
  - Enter in PR modals opens the PR, ctrl+y copies review findings
  - Underline the selected title and right-hand tags in the row colour
  - Select a newly created note after save
  - Inline edit scrolls within the room left of the tags
- [24a31a1](https://github.com/achandrapaul/digest/commit/24a31a1a92f97c579fe90cd53db9b6c7b19cfe4b) feat: isolate brag command, add timeout and note bodies to facts
- [5750ec6](https://github.com/achandrapaul/digest/commit/5750ec68a3f261a043696d0a93859fdd8b7316fe) fix: arrow keys switch git column instead of day
- [6a41d0a](https://github.com/achandrapaul/digest/commit/6a41d0a7a12892fe3c94edf6742e996bf22235e2) feat:pr review, bragging support, more relaxed ui
- [6b0ac47](https://github.com/achandrapaul/digest/commit/6b0ac478e5e1c69ad0a37d392f8f1adccc6e261d) Fetch PRs per repo via batched GraphQL search
  Replace REST search and per-PR gh pr view with one GraphQL search per
  repo, capped by pr_quantity_per_repo (default 15) and ordered by the
  cached pending sort. Add show_daily_commits toggle and drop the gh_*
  command config keys.
- [e052e9e](https://github.com/achandrapaul/digest/commit/e052e9e570535b87543927aed1d6c6373684feae) Include repo and PR number in review note summaries
- [613f87f](https://github.com/achandrapaul/digest/commit/613f87fa02545e320db56d0ee44fa0961bdec402) Initial import of digest
