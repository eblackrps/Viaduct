# Viaduct UI Design Tokens

Viaduct `v3.0.0` standardizes the dashboard visual system around a small shared token set so every page reads like the same application.

## Typography

Tailwind theme tokens:

- `text-display`: primary hero or page-defining statements
- `text-title`: primary page titles and important state titles
- `text-subtitle`: section titles and secondary headings
- `text-body`: default long-form operator copy
- `text-body-sm`: compact supporting copy for cards, tables, and notices
- `text-caption`: labels, badges, and operator kicker text
- `text-mono-sm`: compact monospace diagnostics

## Radius

Standard surface radii:

- `rounded-md`: compact icon wells and small inline containers
- `rounded-lg`: inputs, compact controls, and small cards
- `rounded-xl`: notices, drawers, grouped controls, and medium containers
- `rounded-2xl`: primary surfaces such as cards, tables, shells, and dialogs
- `rounded-full`: pills, badges, and circular actions

## Palette

The operator console uses a slate-based neutral system with:

- brand: `ink`, `steel`, and `brand-400`
- accent: `accent`
- semantic states: `success`, `warning`, `danger`, `info`

Viaduct does not ship a mixed `gray` / `zinc` / `slate` neutral palette. Slate is the canonical neutral family for the dashboard.

## Dark Mode

Dark mode is intentionally not shipped in `v3.0.0`.

The existing dashboard is now treated as a complete light-mode operator surface. Shipping a partial dark theme would create broken contrast, inconsistent chart styling, and incomplete state coverage across the migration, lifecycle, and workspace flows. A future dark-mode effort should be treated as a full product pass rather than an incidental toggle.
