# Docs and releases

- **DR1:** The documentation site is `docs/`, published by GitHub Pages from `main` at https://psyb0t.github.io/servicepack/ with the Just the Docs theme (`remote_theme` pinned in `docs/_config.yml`, colors in `docs/_sass/`, the shared visitor tracker in `docs/_includes/head_custom.html`). The site title equals the README H1.
- **DR2:** Every page has front matter with `title`, a 50 to 170 character `description` and `nav_order`. Links from `docs/` to files outside it use full `https://github.com/psyb0t/servicepack/blob/main/<path>` URLs (`tree/` for directories). README links to the site use full `https://psyb0t.github.io/servicepack/...` URLs.
- **DR3:** After a docs change, run `make docs` and check the built site in `.testing/docs-site`.
- **DR4:** `CHANGELOG.md` entries are `## vX.Y.Z (YYYY-MM-DD)`, a one-line summary, then bullets in the past tense ("Fixed", "Updated", "Added") that say what a downstream must do, if anything.
- **DR5:** Commit subjects for releases start with the version: `vX.Y.Z: <what changed>`.
- **DR6:** A release is an annotated `vX.Y.Z` tag on a commit whose `main` pipeline run passed. The tag message's first line is exactly `vX.Y.Z`, then a blank line, then that version's CHANGELOG section.
