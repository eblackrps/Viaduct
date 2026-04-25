# Viaduct public site

This directory contains the static GitHub Pages site for `viaducthq.com`. The
site is deployed by `.github/workflows/pages.yml` whenever files under `site/`
change on `main`.

## Local preview

```bash
python -m http.server 4173 --directory site
```

Then open `http://localhost:4173`.

## Validation

Run the local version, link, asset, and Pages-workflow check before publishing:

```bash
make site-check
```

After GitHub Pages deploys, verify the public artifact with:

```bash
make site-check-live BASE_URL=https://viaducthq.com
```

The live check fetches the deployed homepage body and fails if it does not
contain the current version, GHCR image tag, and matching GitHub Release link.

## Screenshot assets

The landing page uses product screenshots copied from
`docs/operations/demo/screenshots/` into `site/assets/` so the GitHub Pages
artifact is self-contained. When the dashboard screenshots are refreshed for a
release, copy the updated public-facing captures into `site/assets/` before
publishing the site.

## Release hygiene

Keep the site version, install snippets, release links, and social preview card
aligned with the current release documented in `README.md`, `CHANGELOG.md`, and
`docs/releases/`.
