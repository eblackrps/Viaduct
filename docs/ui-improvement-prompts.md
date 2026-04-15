# Viaduct UI Improvement Prompts — v2.1.0

Produced from a full codebase review on 2026-04-15.  
Each prompt is a self-contained Codex instruction. Apply them in order; later prompts assume earlier ones are complete.

---

## Prompt 1 — Normalize border radii across the entire UI

**Context**  
The current design uses extreme custom border radii (`rounded-[2rem]` = 32 px, `rounded-[1.75rem]` = 28 px, `rounded-[1.8rem]`, `rounded-[1.4rem]`) throughout `web/src/index.css`, `AppShell.tsx`, `SidebarNav.tsx`, `PageHeader.tsx`, and most component files. This produces a "bubbly" aesthetic that is inconsistent with a professional operator control plane. The Tailwind config ships a `panel` box-shadow but no normalized radius token.

**Task**  
1. In `web/src/index.css` update the `.panel` component class:  
   - Change `rounded-[2rem]` → `rounded-2xl` (16 px)  
2. Update `.panel-muted`: change `rounded-[1.75rem]` → `rounded-xl` (12 px)  
3. In `AppShell.tsx`:  
   - Brand card inner gradient div: `rounded-[1.8rem]` → `rounded-xl`  
   - Info callout divs that use `panel-muted p-4`: already inherits from `.panel-muted`, remove any inline `rounded-*` override  
   - Main content wrapper: remove hardcoded `rounded-[...]` if present  
4. In `SidebarNav.tsx`:  
   - Nav link `className`: `rounded-[1.4rem]` → `rounded-xl`  
   - Icon span: `rounded-2xl` stays (it is already a standard value)  
5. In `PageHeader.tsx`:  
   - Outer `header`: `rounded-[1.75rem]` → `rounded-xl`  
6. In `TopBar.tsx`:  
   - All metric surface divs and inline info panels: audit and replace any remaining `rounded-[...]` with `rounded-xl` or `rounded-2xl`  
7. Run `grep -r "rounded-\[" web/src/` and replace every remaining custom value with the nearest standard Tailwind step (`rounded-lg`=8 px, `rounded-xl`=12 px, `rounded-2xl`=16 px).

**Acceptance criteria**  
- `grep -r "rounded-\[" web/src/` returns zero results  
- `npm run build` in `web/` succeeds with no TypeScript errors

---

## Prompt 2 — Compact sidebar navigation items (remove per-item descriptions)

**Context**  
`web/src/components/navigation/SidebarNav.tsx` renders each nav item as a tall card with icon (40×40 px) + title + description line. With 10 nav items across 5 groups this produces a sidebar that requires ~900 px of vertical space just for navigation, forcing most real viewports to scroll before seeing any page content.

**Task**  
1. In `SidebarNav.tsx`, refactor each nav item `<a>` to a compact row:  
   - Remove the `<span className="mt-1 block text-xs leading-5 ...">` description line entirely  
   - Reduce the icon container: `h-10 w-10` → `h-8 w-8`, keep `rounded-xl` and the active/inactive colour classes  
   - Reduce icon size inside: `h-4 w-4` → `h-4 w-4` (no change, already small)  
   - Change the outer `<a>` padding: `px-4 py-3` → `px-3 py-2.5`  
   - Change the `<span className="block text-sm font-semibold">` to stay as-is (label only, no mt-1 sibling anymore)  
   - Remove `items-start` from the flex container; use `items-center` now that there is only one text line  
2. In `SidebarNav.tsx`, remove the `<section className="panel-muted p-4">` "Operator path" block at the bottom of the component. This content is already covered by the "Default flow" callout in the brand panel and in docs.  
3. Ensure the `aria-current` and focus-ring classes are preserved.

**Acceptance criteria**  
- Each nav item renders as a single compact row (icon + label) with no description line  
- No "Operator path" section below the nav groups  
- `npm run build` succeeds

---

## Prompt 3 — Slim the sidebar brand panel

**Context**  
`web/src/layouts/AppShell.tsx` renders a large `<aside>` containing:  
1. A panel with a brand card (gradient background, 4-line tagline) + two `.panel-muted` callout boxes ("Default flow" and "Shared truth")  
2. A second panel containing the nav  

The "Default flow" and "Shared truth" info boxes repeat content that lives in the docs and the operator path section. They consume ~200 px before the user even reaches navigation.

**Task**  
1. In `AppShell.tsx`, remove the two `<div className="panel-muted p-4">` callout blocks ("Default flow" and "Shared truth").  
2. Reduce the brand card inner padding: `px-6 py-6` → `px-5 py-4`  
3. Shorten the brand tagline copy to a single sentence (≤ 12 words):  
   - Replace the current 30-word tagline with: `"Discover, plan, and execute migrations from one tenant-scoped control plane."`  
4. Remove the `<div className="mt-5 grid gap-3">` wrapper that contained the two callout divs.  
5. Merge the brand card and nav into a single `<div className="panel p-4 space-y-4">` so the aside only contains one panel instead of two (reduces visual noise and extra border lines).

**Acceptance criteria**  
- Aside contains one panel with brand header + nav  
- No "Default flow" or "Shared truth" callouts  
- `npm run build` succeeds

---

## Prompt 4 — Simplify the TopBar: remove static noise badges and metric grid

**Context**  
`web/src/components/navigation/TopBar.tsx` renders:  
- A title + description + auth badges row  
- A row of static info badges: "REST API + shared store", "Tenant-scoped visibility" — these never change and carry no operator signal  
- A 5-column metric grid (tenant context, workloads, active migrations, pending approvals, recommendations)  

The metric grid duplicates data already shown on the Dashboard page. The result is ~280 px of fixed chrome before any page content. On every page.

**Task**  
1. Remove the two static `.panel-muted` info-badge divs:  
   - `REST API + shared store`  
   - `Tenant-scoped visibility`  
2. Remove the entire `<div className="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-[...]">` metric grid block from `TopBar`.  
3. Keep: page title, description, `tenantId` badge, auth mode badge, persistence label badge, refresh button, sign-out button.  
4. Lay out the remaining elements as a single horizontal flex row that wraps on small screens:  
   ```
   [page-title block]          [actions: auth-badge? | refresh | sign-out?]
   [description text]
   ```  
5. Reduce top-level panel padding: `p-5` → `p-4`.  
6. Remove `TopBarStatusItem` interface and `statusItems` prop entirely since they are no longer rendered.  
7. Update `AppShell.tsx` to no longer pass `statusItems`, `lastDiscoveryAt` to `TopBar` (those props can be removed from `AppShellProps` too if not used elsewhere). Keep `lastDiscoveryAt` as an optional display in the auth badge area as a single small text line.

**Acceptance criteria**  
- TopBar renders as a compact two-line header (title + description + action buttons)  
- No metric grid in the TopBar  
- No static "REST API…" or "Tenant-scoped…" badges  
- TypeScript compilation succeeds with no unused-prop errors

---

## Prompt 5 — Fix redundant status badge labels in metric cards and signal rows

**Context**  
Two places repeat the same text as both a label and a StatusBadge:

**A. `web/src/features/dashboard/DashboardPage.tsx` — metric cards section:**
```tsx
<p className="text-xs uppercase tracking-[0.22em] text-slate-500">{card.label}</p>
<p className="mt-3 font-display text-3xl text-ink">{card.value}</p>
<div className="mt-3">
  <StatusBadge tone={card.tone}>{card.label}</StatusBadge>  {/* ← duplicate */}
</div>
```
The `StatusBadge` says "Workloads" beneath a card already titled "Workloads". It adds zero information.

**B. `DashboardPage.tsx` — `SignalRow` component:**
```tsx
<p className="font-semibold text-ink">{label}</p>
<StatusBadge tone={badgeTone}>{label}</StatusBadge>  {/* ← duplicate */}
```
The badge repeats the row label. It should show a meaningful status word instead.

**Task**  
1. In `DashboardPage.tsx`, remove the `<div className="mt-3"><StatusBadge ...>{card.label}</StatusBadge></div>` from the metric card render loop.  
2. In `SignalRow`, change the badge content from `{label}` to a status word derived from `badgeTone`:  
   ```tsx
   const toneLabel: Record<StatusTone, string> = {
     success: "Healthy",
     warning: "Attention",
     danger: "Critical",
     info: "Active",
     neutral: "None",
     accent: "Active",
   };
   // ...
   <StatusBadge tone={badgeTone}>{toneLabel[badgeTone]}</StatusBadge>
   ```  
3. Apply the same audit to `TopBar.tsx` metric surfaces (already removed in Prompt 4) and any other location where a `StatusBadge` directly repeats adjacent text — fix or remove each instance.

**Acceptance criteria**  
- No `StatusBadge` renders text identical to its sibling label element  
- `npm run build` succeeds

---

## Prompt 6 — Remove the outer AppShell page-content wrapper panel

**Context**  
`web/src/layouts/AppShell.tsx` wraps all page content in:
```tsx
<div className="panel p-6 lg:p-7">{children}</div>
```
Every page then renders its own `PageHeader` (with border + background) and `SectionCard` panels inside this outer panel, creating 3–4 layers of white boxes, borders, and padding. The nesting adds visual weight without structure.

**Task**  
1. In `AppShell.tsx`, change the `{children}` wrapper from:  
   ```tsx
   <div className="panel p-6 lg:p-7">{children}</div>
   ```  
   to:  
   ```tsx
   <div className="space-y-5">{children}</div>
   ```  
2. Each page's `PageHeader` and `SectionCard` already have their own `panel` / `panel-muted` styling — they will now sit directly in the page flow without a redundant outer box.  
3. Verify the error banner in `AppShell.tsx` still renders correctly in the `<main>` space-y flow.

**Acceptance criteria**  
- Page content is no longer wrapped in an outer `.panel` div  
- All pages still render their own `PageHeader` and `SectionCard` panels correctly  
- `npm run build` succeeds

---

## Prompt 7 — Fix duplicate icons in navigation

**Context**  
`web/src/app/navigation.ts` assigns `LayoutDashboard` to both the `/workspaces` (Pilot Workspaces) and `/dashboard` (Overview) routes. Having identical icons for different sections is visually ambiguous and looks like a copy-paste error.

**Task**  
Replace icons to make each route visually distinct:

| Route | Current Icon | New Icon |
|-------|-------------|----------|
| `/workspaces` | `LayoutDashboard` | `FolderKanban` |
| `/dashboard` | `LayoutDashboard` | `LayoutDashboard` (keep) |
| `/inventory` | `Database` | `Server` |
| `/migrations` | `Waypoints` | `Waypoints` (keep) |
| `/lifecycle` | `Coins` | `TrendingUp` |
| `/policy` | `ShieldCheck` | `ShieldCheck` (keep) |
| `/drift` | `Activity` | `GitCompare` |
| `/reports` | `FileText` | `FileText` (keep) |
| `/settings` | `Settings` | `Settings` (keep) |
| `/graph` | `GitBranch` | `Network` |

1. In `navigation.ts`, update the import list to include `FolderKanban`, `Server`, `TrendingUp`, `GitCompare`, `Network`.  
2. Remove `Coins`, `Activity`, `GitBranch`, `Database` from the import if no longer used elsewhere.  
3. Update each route's `icon` field per the table above.

**Acceptance criteria**  
- No two routes share the same icon component  
- `npm run build` succeeds with no unused-import warnings

---

## Prompt 8 — Add mobile sidebar drawer

**Context**  
`AppShell.tsx` uses `xl:grid-cols-[320px_1fr]` which only shows the sidebar at the `xl` breakpoint (≥ 1280 px). On laptops and tablets the sidebar collapses above the main content, creating a poor stacked layout.

**Task**  
1. Create `web/src/components/navigation/MobileSidebarDrawer.tsx`:  
   - Renders a fixed full-height `<aside>` with the same `SidebarNav` content  
   - Controlled by an `open` boolean prop and an `onClose` callback  
   - Backdrop: `fixed inset-0 bg-ink/40 z-40` (closes on click)  
   - Drawer: `fixed inset-y-0 left-0 z-50 w-72 bg-white shadow-xl p-4 flex flex-col gap-4 transition-transform`  
   - Transform: `translate-x-0` when open, `-translate-x-full` when closed  
   - Include a close `×` button inside the drawer  
2. In `AppShell.tsx`:  
   - Add `mobileNavOpen` state (default `false`)  
   - Add a hamburger button (`Menu` icon from lucide-react) visible only below `xl:` (`xl:hidden`) in the `<main>` area, positioned in the top-left of the top bar row  
   - Render `<MobileSidebarDrawer>` controlled by that state  
   - Hide the `<aside>` on smaller screens: add `hidden xl:block` to the aside  
3. Ensure `aria-label="Open navigation"` on the hamburger button and `aria-label="Close navigation"` on the close button.

**Acceptance criteria**  
- At viewport width < 1280 px, a hamburger button appears and the drawer opens/closes correctly  
- At viewport width ≥ 1280 px, the static sidebar shows and the hamburger is hidden  
- No layout shift on desktop  
- `npm run build` succeeds

---

## Prompt 9 — Improve AppShell error display

**Context**  
`AppShell.tsx` renders runtime errors as a bare paragraph:
```tsx
{error && <p className="panel-muted border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">{error}</p>}
```
There is no icon, no title, and no way to dismiss the error.

**Task**  
1. Replace the inline error `<p>` in `AppShell.tsx` with an `ErrorBanner` component defined inline or extracted to `web/src/components/primitives/ErrorBanner.tsx`:  
   ```tsx
   interface ErrorBannerProps {
     message: string;
     onDismiss?: () => void;
   }
   export function ErrorBanner({ message, onDismiss }: ErrorBannerProps) {
     return (
       <div
         role="alert"
         className="flex items-start gap-3 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700"
       >
         <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-rose-500" />
         <p className="flex-1 leading-5">{message}</p>
         {onDismiss && (
           <button
             type="button"
             onClick={onDismiss}
             aria-label="Dismiss error"
             className="ml-auto shrink-0 rounded p-0.5 hover:bg-rose-100"
           >
             <X className="h-4 w-4" />
           </button>
         )}
       </div>
     );
   }
   ```  
2. In `AppShell.tsx`, import and use `<ErrorBanner message={error} />`.  
3. Import `AlertTriangle` and `X` from `lucide-react`.

**Acceptance criteria**  
- Error banner shows icon + message + dismiss button  
- `role="alert"` present for accessibility  
- `npm run build` succeeds

---

## Prompt 10 — Tighten typography hierarchy

**Context**  
The current design has two competing large headings on every page:  
- `TopBar.tsx` renders the page title at `font-display text-3xl` inside `.panel`  
- `PageHeader.tsx` renders a *second* title for the same page at `font-display text-4xl`  

With the TopBar simplification from Prompt 4, the TopBar title is still `text-3xl`. The `PageHeader` title should be the primary heading (`text-2xl`) and the TopBar should be a compact navigation label (`text-lg` or `text-base font-semibold`).

**Task**  
1. In `TopBar.tsx`, change the page title size: `font-display text-3xl` → `text-base font-semibold text-ink` (it is a navigation label, not a page heading).  
2. In `TopBar.tsx`, change the description below the title: `text-sm text-slate-600` → `text-xs text-slate-500` (reduce visual weight).  
3. In `PageHeader.tsx`, change the title: `font-display text-4xl` → `font-display text-2xl` (this becomes the true H1 for the page).  
4. In `PageHeader.tsx`, change the description: `text-sm leading-6` → `text-sm leading-5` (tighten line-height for shorter blocks).  
5. In `SectionCard.tsx`, the section title `font-display text-2xl` is already at the right size; no change needed.  
6. Audit `AppShell.tsx` brand card: `font-display text-4xl` for the "Viaduct" wordmark stays — this is a brand element, not a content heading.

**Acceptance criteria**  
- Each page has exactly one primary heading (the `PageHeader` title at `text-2xl`)  
- The TopBar shows a compact navigation label, not a competing page heading  
- `npm run build` succeeds

---

## PR Review Prompt

> You are a senior PR engineer reviewing a UI polish pull request for **Viaduct**, an operator control plane for virtualization migration. The branch is `ui/polish-v2.1.0`. The changes are a series of 10 UI improvements applied to `web/src/` only — no backend Go code was changed.
>
> **Review criteria — evaluate each of the following:**
>
> 1. **Correctness** — Do the changes do exactly what the prompt specified? Are there any accidental regressions (removed functionality, broken prop interfaces, missing imports)?
> 2. **TypeScript safety** — Do all components still have correct prop types? Were any props removed from interfaces without updating all call sites? Are there any `any` types introduced?
> 3. **Accessibility** — Are `aria-label`, `aria-current`, `role` attributes preserved or improved? Did any changes remove keyboard navigation or focus-ring styles?
> 4. **Visual consistency** — Are the border radius values now consistent (`rounded-xl` / `rounded-2xl` only)? Are there any remaining `rounded-[...]` custom values? Are icon assignments unique per route?
> 5. **Redundancy removal** — Are all duplicate badge labels gone? Are the static TopBar info badges removed? Is the AppShell outer panel wrapper gone?
> 6. **Mobile sidebar** — Does the `MobileSidebarDrawer` correctly use `xl:hidden` / `hidden xl:block` to prevent both the static and drawer sidebars from showing simultaneously?
> 7. **Build integrity** — Does `npm run build` in `web/` complete with zero TypeScript errors and zero console warnings?
> 8. **Scope discipline** — Did any change exceed the stated prompt scope (e.g., touching Go backend, modifying test files, restructuring page logic)?
>
> **Output format:**  
> For each of the 8 criteria, mark **PASS**, **FAIL**, or **WARN** with a one-sentence explanation. If any criterion is FAIL, list the exact file and line number to fix.

---

*End of prompts document.*
