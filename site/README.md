# Public Site

This directory contains the standalone static site for GitHub Pages. It is separate from the product dashboard in `web/` so the public site can ship without the application build.

## Files

- `index.html`: landing page
- `404.html`: custom not-found page
- `styles.css`: shared site stylesheet
- `favicon.svg`: site icon
- `social-card.svg`: social-preview image for metadata
- `robots.txt`: crawler policy
- `sitemap.xml`: Pages sitemap

## Local Preview

For a quick check, open `site/index.html` directly in a browser.

If you prefer to serve it over HTTP:

```powershell
python -m http.server 4173 --directory site
```

Then open `http://localhost:4173`.

## Deployment

GitHub Pages deploys this directory through `.github/workflows/pages.yml`. The workflow uploads `site/` whenever `main` changes in this directory, or when the workflow is run manually.

The Pages deployment is separate from GitHub releases. Release tags package the product artifacts; the public site publishes from the `main` branch workflow.

## Domain Notes

The checked-in `CNAME` targets `viaducthq.com`. Asset paths in this site are relative so the same content works on both the custom domain and the default project Pages URL.

If Pages settings ever need to be rebuilt manually:
- keep the publishing source on `GitHub Actions`
- keep the custom domain aligned with `site/CNAME`
- verify the domain before enabling HTTPS enforcement

## Editing Guidance

- Keep the site static and GitHub Pages-compatible.
- Use concise operator- and buyer-readable language.
- Keep copy honest about current product maturity and controlled operator workflows without slipping into roadmap jargon.
- Prefer changes in `site/` over changing the application dashboard just to support marketing copy.
