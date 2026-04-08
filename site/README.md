# Public Site

This directory contains the standalone static site for GitHub Pages. It is separate from the product dashboard in `web/` so the public marketing surface can deploy cleanly without pulling in the application build.

## Files
- `index.html`: primary landing page
- `404.html`: lightweight custom not-found page
- `styles.css`: shared site styles
- `favicon.svg`: simple site icon

## Local Preview

For a quick visual check, open `site/index.html` directly in a browser.

If you prefer to serve it over HTTP, use any static file server. For example, if Python is installed, run this from the repository root:

```powershell
python -m http.server 4173 --directory site
```

Then open `http://localhost:4173`.

## Deployment

GitHub Pages deploys this directory through `.github/workflows/pages.yml`. The workflow uploads `site/` as the Pages artifact whenever `main` changes in this directory, or when the workflow is run manually.

This Pages deployment is separate from GitHub Releases. Release tags package the product artifacts; the public site publishes from the `main` branch workflow.

The repository's default Pages URL will be:

`https://eblackrps.github.io/viaduct/`

All asset links in this site are relative so the same build works both on the default project Pages URL and on a custom domain.

## Custom Domain

For the current repository owner, make `viaducthq.com` the primary domain and point `www.viaducthq.com` at GitHub Pages.

1. In GitHub, open `Settings -> Pages` for `eblackrps/viaduct`.
2. Set the publishing source to `GitHub Actions` if it is not already set.
3. In the `Custom domain` field, enter `viaducthq.com`.
4. In DNS, configure either:

| Host | Type | Value |
| --- | --- | --- |
| `@` | `ALIAS` or `ANAME` | `eblackrps.github.io` |

Or the explicit GitHub Pages apex records:

| Host | Type | Value |
| --- | --- | --- |
| `@` | `A` | `185.199.108.153` |
| `@` | `A` | `185.199.109.153` |
| `@` | `A` | `185.199.110.153` |
| `@` | `A` | `185.199.111.153` |
| `@` | `AAAA` | `2606:50c0:8000::153` |
| `@` | `AAAA` | `2606:50c0:8001::153` |
| `@` | `AAAA` | `2606:50c0:8002::153` |
| `@` | `AAAA` | `2606:50c0:8003::153` |
| `www` | `CNAME` | `eblackrps.github.io` |

With `viaducthq.com` set as the custom domain in GitHub, GitHub Pages should redirect `www.viaducthq.com` to the apex domain once both record sets resolve correctly.

## Verification And HTTPS

For takeover protection, verify the domain in the account or organization Pages settings and keep the TXT verification record in DNS. After GitHub finishes issuing the certificate, enable `Enforce HTTPS` in the repository Pages settings.

## Notes

- Do not use wildcard DNS records for this site.
- A `CNAME` file is not required for this workflow because custom domains are configured through GitHub Pages settings when deploying via GitHub Actions.
