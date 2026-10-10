# Vanguard Docs Site

This directory hosts the VitePress site for Vanguard.

## Local development

```bash
nvm use 22
cd docs
npm install
npm run docs:dev
```

## Build

```bash
nvm use 22
cd docs
npm install
npm run docs:build
```

## Notes

- Site content is maintained directly under `docs/` — the engineering documents
  that predate the site (`charter.md`, `config.md`, `install.md`,
  `ci-integration.md`, `rules/*`) are canonical pages of the site, not
  auto-generated output.
- `docs/rules/*.md` must stay 1-1 with the binary's rule catalog
  (`internal/cli/rulesdocs_test.go` enforces it in `make ci`) — do not rename
  or move rule pages.
- `ignoreDeadLinks` is enabled in `.vitepress/config.mts` because engineering
  pages legitimately reference repository paths outside the site
  (`reports/*`, `.vanguard.example.yaml`, `testdata/*`).
