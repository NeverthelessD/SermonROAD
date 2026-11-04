# SermonROAD — notes for Claude

**Read `PROJECT_NOTES.md` first** (structure, decisions, progress checklist) and update it at the end of every task.
SermonROAD is a personal macOS (Apple Silicon) sermon record / preparation app for **Nevertheless_D**, a Reformed preacher who is
not a programmer. Talk to them in Korean, in plain words. Never ask them to use Terminal. Never mention GitHub (or the repo name)
in anything the user sees (app screens, update text, guide image).

It follows the user's ROAD app pattern (see the `road-mac-app` skill and the MSROAD repo): a real, visible Dock app
(Swift shell `app/shell/main.swift`, WKWebView window) + Go engine (`engine/`, web UI embedded from `engine/web`) that runs as the
app's child (app → supervisor → engine) only while the app is open. Never reintroduce hidden helpers or SIGTERM-ignoring.
A separate CLI tool `Contents/Resources/sermonroad-pdftext <pdf> <out.json>` (app/pdftext/main.swift, PDFKit + Vision OCR) reads library PDFs;
it is not the bundle executable on purpose, so running it never puts an icon in the Dock.

The app can only be built on a Mac: pushing changes under app/, engine/, VERSION, scripts/testdata or the mac scripts triggers
`.github/workflows/build-mac.yml`, which builds, ad-hoc signs, runs real-Mac smoke tests (`scripts/smoke-mac.sh`) and force-pushes
results to the `ci-results` branch (read via git: `git fetch origin +refs/heads/ci-results:refs/remotes/origin/ci-results`).
Release with `scripts/release.sh --notes …` only when build=ok and smoke=ok; it also refreshes `download/SermonROAD.zip`.
In tests use `pkill -x`, never `pkill -f sermonroad-engine`. Linux dev run: see PROJECT_NOTES.md (시험하는 법).
