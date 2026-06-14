# Dashboard Design System

A reference document describing the visual language, layout structure, and UI components used in the "Codename.com" analytics dashboard.

---

## 1. Overview

The dashboard follows a clean, minimal, light-themed admin UI style with a single strong accent color used for emphasis (primary actions, highlights, key metrics). The layout uses a fixed left navigation rail, a top header/toolbar, and a responsive grid of cards for reports, charts, and tables.

---

## 2. Color Palette

| Role | Color | Hex (approx) | Usage |
|---|---|---|---|
| Background (app) | Light gray | `#F4F4F6` | Page background |
| Surface / Card | White | `#FFFFFF` | Cards, panels, sidebar items |
| Primary accent | Pink / Crimson | `#E8326E` | Active sidebar icon, primary buttons, highlighted card, badges, key data labels |
| Primary dark | Near-black navy | `#1E1E24` | "Best deal" card, dark CTA buttons, primary text |
| Text primary | Dark gray | `#1A1A1E` | Headings, large numbers |
| Text secondary | Mid gray | `#8C8C99` | Labels, captions, subtitles |
| Text muted | Light gray | `#B8B8C2` | Placeholder text, disabled states |
| Success / Positive | Green | `#3DCB6C` | Positive percentage badges (e.g. +7.9%) |
| Error / Negative | Red | `#F04438` | Negative percentage badges, alert dots |
| Border / Divider | Pale gray | `#E9E9ED` | Card borders, separators |
| Chart accent (secondary) | Soft pink | `#FBD9E4` | Bar chart fills, chart highlight zones |
| Avatar colors (varied) | Multi | Blue, teal, yellow, pink | User avatar backgrounds for differentiation |

**Usage principle:** White/neutral surfaces dominate; the pink accent is reserved for the single most important data point per section (e.g., revenue change, active nav icon, "Details" badge counts), and the dark navy is reserved for one "hero" highlight card per view (e.g., "Best deal").

---

## 3. Typography

- **Font family:** A geometric/grotesque sans-serif (e.g., Inter, SF Pro, or similar).
- **Hierarchy:**
  - **Display / Hero numbers** (e.g., `$528,976.82`): Bold, ~32–36px, dark text.
  - **Section titles** (e.g., "New report", "Revenue"): Medium-bold, ~16–18px.
  - **Card labels** (e.g., "Deals", "Value", "Win rate"): Regular, ~12–13px, secondary gray.
  - **Body / table rows**: Regular, ~13–14px.
  - **Captions / micro-labels** (percentages, dates, counts): ~11–12px, often bolded inside badges.
- **Letter spacing:** Slightly tight on large numerals; normal elsewhere.
- **Numeric emphasis:** Large monetary values use tabular figures for alignment in lists/tables.

---

## 4. Layout Structure

### 4.1 Grid
- **Left Sidebar:** Fixed width (~80px icon rail + collapsible labeled panel), full viewport height.
- **Main Content:** Fluid width, padded (~20–24px), organized into a responsive multi-column card grid.
- **Header Row:** Spans the main content width; contains collaborator avatars, search bar, and action icons.

### 4.2 Spacing
- Base spacing unit: **8px**.
- Card padding: **16–20px**.
- Gap between cards: **12–16px**.
- Border radius: **12–16px** on cards, **8px** on smaller chips/badges, **full/circular** on avatars and icon buttons.

---

## 5. Components

### 5.1 Sidebar Navigation
- Vertical icon rail with a circular logo mark at the top.
- Icons represent: Home/dashboard, calendar, goals (highlighted in pink — active state), inbox, integrations, settings.
- Active item indicated by a filled pink rounded-square background.
- Below the icon rail: a secondary panel listing "Starred", "Recent", "Sales list", "Goals", "Dashboard", "Reports", with nested folder-style lists ("Shared with me", "My reports") and small red notification badges on certain items.
- Bottom of sidebar: user/profile icon and a settings gear icon.

### 5.2 Top Header / Toolbar
- Logo + workspace name dropdown (left).
- Global search bar (centered/left, with placeholder text like "Try searching 'insights'").
- Right-aligned icons: filter/share, download, and a circular "add" button in pink.
- Collaborator avatar stack with names, plus a "+" add-collaborator icon.

### 5.3 Stat / Hero Card ("Revenue")
- Large title ("New report") in muted gray, page-title style.
- Below it, a bold large numeral for the primary metric.
- A small pill badge (green or red) showing percentage change, with an up/down arrow icon.
- Comparison subtext (e.g., "vs prev. $501,641.73 ... ") with a date-range dropdown.
- A timeframe toggle switch with a date-range label, right-aligned.

### 5.4 KPI Mini-Cards
- Small rounded cards arranged in a row, each containing:
  - A label (e.g., "Top sales", "Deals", "Value", "Win rate").
  - A large bold number/value.
  - A small colored badge showing change/trend.
  - Optional avatar (for "Top sales" — shows top performer).
- One card uses the **dark navy "hero" treatment** ("Best deal") with a star icon, value, and a circular arrow button — visually distinct from the light cards around it.

### 5.5 Source / Channel Breakdown Bar
- A horizontal row of compact pill items, each showing:
  - A small colored icon/avatar (platform logo or initials).
  - A monetary value.
  - A percentage badge.
- Ends with a dark "Details" button (rounded pill, dark background, white text).

### 5.6 Data Table (Sales/Leads)
- Column headers: Sales, Revenue, Leads, KPI, W/L (and similar).
- Rows show: avatar + name, revenue figure, leads count (in a circular badge), KPI decimal, and a win/loss percentage with colored circular indicator (red/green dot).
- Alternating emphasis: some rows highlighted with colored badges (e.g., "Top sales", "Sales streak", "Top review" tags shown as small pill labels with emoji/icon).

### 5.7 Ranked List Card (e.g., "Deals by Referrer")
- Header row with title + filter icon/button.
- List items each showing:
  - Small colored square/icon (platform identity color).
  - Platform name.
  - Dollar value (right-aligned).
  - Percentage share badge (light gray pill).

### 5.8 Donut/Segment Chart Card
- Circular segmented chart (donut) showing proportional breakdown.
- Platform icon avatars arranged around or below the chart.
- Caption below explaining the chart (e.g., "Deals amount by referrer category").

### 5.9 Featured Platform Card ("Platform Value")
- Large pink-accent card (full bleed pink background) as a featured/spotlight tile.
- Header shows platform name + dropdown ("Dribbble ▾") and a segmented toggle (Revenue / Leads / W/L).
- Inside: large white numeral for the metric, plus secondary stats (Average monthly, Leads ratio, Win/Loss ratio) in smaller white/light text.

### 5.10 Bar Chart
- Adjacent to the featured card: a vertical bar chart with light pink/gray bars.
- One bar highlighted with a pink callout bubble showing its exact value (e.g., "$11,035").
- Smaller floating value bubbles above other bars (e.g., "$6,901", "$9,288").
- X-axis labeled with month abbreviations.

### 5.11 Line Chart ("Sales Dynamic")
- Multi-series line/area chart with smooth curves.
- Pink line as the primary series, secondary muted line for comparison.
- Small circular avatar markers placed directly on the chart line at specific data points to denote events/contributors.
- X-axis with day numbers.

### 5.12 Work-with-Platforms Cards
- Set of 2–3 compact cards, each representing a platform (Dribbble, Instagram, Google):
  - Platform icon (colored circle).
  - Large percentage figure.
  - Supporting revenue value and a secondary stat (e.g., leads, average rating with star icon).

---

## 6. Iconography & Avatars
- **Icons:** Simple line/outline icons (16–20px), consistent stroke weight, used for navigation, actions (search, filter, download, share, add), and chart controls.
- **Avatars:** Circular, ~24–32px, filled with brand-colored backgrounds and either initials or platform logos (Dribbble, Instagram, Google, Behance).
- **Status dots:** Small circular indicators (green/red) used to denote win/loss or positive/negative trend, often overlapping the bottom-right of an avatar or value.

---

## 7. Interaction Patterns
- **Dropdowns:** Used for date ranges, platform selectors, and folder navigation — indicated with a chevron-down icon.
- **Toggles:** Pill-shaped switches for timeframe and chart-metric selection (Revenue / Leads / W/L).
- **Badges/Pills:** Used universally for percentage changes, counts, and category tags — rounded-full shape, small font, colored background matching semantic meaning (green = positive, red = negative/alert, gray = neutral).
- **Hover/Active states:** Active sidebar items and selected toggle segments use the pink accent or dark fill to indicate selection.

---

## 8. Design Principles Summary
1. **Single accent rule** — pink is used sparingly but consistently for the most important call-to-action or metric per section.
2. **Card-based modularity** — every data unit lives in its own rounded white card, enabling a flexible grid layout.
3. **Data-first typography** — numbers are the visual hierarchy anchors; labels stay small and muted.
4. **Soft, rounded geometry** — generous border radii across cards, buttons, badges, and avatars create a friendly, modern feel.
5. **Light, airy background** — neutral gray app background keeps focus on white card surfaces and accent colors.
