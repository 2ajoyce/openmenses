# TODO — openmenses

This file tracks implementation tasks. Items are grouped by phase and step.
Check off tasks as they are completed.

---

## Phase 6: UI Expansion

Phase 6 adds advanced visualizations, export tools, a clinician summary, and accessibility improvements. All features compose existing Connect-RPC endpoints — no new proto definitions or backend changes are needed. The phase is split into three independently shippable sub-phases.

### Background / Key Decisions

- **No new proto/RPC endpoints**: All Phase 6 features compose existing RPCs (`ListCycles`, `GetCycleStatistics`, `ListTimeline`, `ListPredictions`, `ListInsights`, `CreateDataExport`, `ListMoodObservations`, `ListMedications`, `GetUserProfile`).
- **Recharts for charting**: SVG-based React charting library (~45KB gzipped). Supports theming via CSS custom properties, touch-friendly, works offline.
- **CSV export is client-side**: Converting JSON export to CSV is a presentation concern, not domain logic. Handled in the UI layer using enum labels from `lib/enums.ts`.
- **Calendar heatmap is custom HTML/CSS**: No additional library needed — built with a simple grid layout.
- **All CSS in `theme.css`**: Per `UI_Design_Guidelines.md`, all custom styles go in `ui/src/app/theme.css` using `--om-` design tokens and `om-` utility classes. No CSS-in-JS, modules, or Tailwind.
- **Dark mode mandatory**: Every new color token must have a `.dark` counterpart.
- **Clinician summary uses `window.print()`**: Printable page with `@media print` styles. Works in mobile WebViews via the OS print dialog (includes "Save as PDF").

---

## Sub-Phase 6A: Export Tools + Accessibility Foundation

### Step 1: Export Page (VERIFIED)

Wire up the existing `CreateDataExport` RPC to a download UI. Add CSV conversion client-side.

- [x] Create `ui/src/features/export/ExportPage.tsx` — page with "Export JSON" and "Export CSV" buttons
  - Call `CreateDataExport` RPC on click
  - Use `Blob` + `URL.createObjectURL` + programmatic `<a>` click to trigger download
  - Show loading state during export
  - Show success/error feedback
- [x] Create `ui/src/features/export/csvConverter.ts` — pure function converting JSON export payload to CSV strings
  - Flatten each record type (bleeding, symptoms, moods, medications, medication events) into rows
  - Use human-readable enum labels from `lib/enums.ts`
  - Timestamps in ISO 8601 format
  - Handle empty optional fields gracefully
- [x] Add route `/export/` → `ExportPage` in `ui/src/app/App.tsx`
- [x] Update `ui/src/pages/SettingsPage.tsx` — link "Export Data" item to navigate to `/export/`
- [x] Create `ui/src/features/export/ExportPage.test.tsx` — renders buttons, verifies click handlers
- [x] Create `ui/src/features/export/csvConverter.test.ts` — verifies CSV output format with known JSON input
- [x] Run `make ui-lint` — must pass
- [x] Run `make ui-test` — must pass

---

### Step 2: Accessibility Improvements (VERIFIED)

Systematic a11y pass across all existing and new components.

- [x] Add `.om-sr-only` utility class to `ui/src/app/theme.css` (visually hidden, screen-reader accessible)
- [x] Audit color contrast for WCAG AA compliance (4.5:1 normal text, 3:1 large text); adjust dark mode tokens if needed
- [x] `ui/src/features/timeline/TimelinePage.tsx`:
  - Add `aria-label` to filter chips
  - Add `role="feed"` to timeline list
  - Add `aria-live="polite"` to loading states
- [x] `ui/src/features/cycle/CyclesPage.tsx`:
  - Add `role="region"` + `aria-label` to each section
  - Add `aria-label` to stat values
- [x] `ui/src/components/EnumSelector.tsx`:
  - Add `role="radiogroup"` or `role="group"` with `aria-label`
- [x] `ui/src/components/ConfirmDialog.tsx`:
  - Add `role="alertdialog"`, `aria-labelledby`, `aria-describedby`
- [x] `ui/src/components/EmptyState.tsx`:
  - Add `role="status"`
- [x] All form components (`BleedingForm`, `SymptomForm`, `MoodForm`):
  - Add `aria-describedby` for validation error messages
- [x] Run `make ui-lint` — must pass
- [x] Run `make ui-test` — must pass

---

## Sub-Phase 6B: Visualizations

### Step 3: Install Recharts + Chart Infrastructure (VERIFIED)

- [x] Add `recharts` dependency to `ui/package.json`
- [x] Run `npm install` — must succeed
- [x] Add chart design tokens to `ui/src/app/theme.css` (with dark mode variants):
  - [x] `--om-color-chart-axis`
  - [x] `--om-color-chart-grid`
  - [x] `--om-color-chart-tooltip-bg`
  - [x] `--om-color-chart-tooltip-border`
- [x] Add `.om-chart-container` utility class (min-height, responsive width, padding)
- [x] Create `ui/src/components/ChartContainer.tsx` — wrapper around Recharts `ResponsiveContainer`
  - [x] Handle empty-data state (render `EmptyState` component)
  - [x] Provide consistent card shell
- [x] Create `ui/src/components/ChartContainer.test.tsx` — empty state vs children rendering
- [x] Run `make ui-lint` — must pass
- [x] Run `make ui-test` — must pass

---

### Step 4: Cycle Length Trend Chart (VERIFIED)

Line chart showing cycle length over time on the Cycles page.

- [x] Create `ui/src/features/cycle/CycleLengthChart.tsx`:
  - Recharts `LineChart` with `ResponsiveContainer`
  - X-axis: cycle index (1, 2, 3, ...)
  - Y-axis: length in days
  - Dashed horizontal reference line at average cycle length
  - Line stroke uses `--om-color-primary`
  - Tooltip shows cycle dates and length
  - Only renders with 2+ completed cycles
  - Wrap in `role="img"` with descriptive `aria-label`
  - Include `.om-sr-only` text summary below chart for screen readers
- [x] Create `ui/src/features/cycle/CycleLengthChart.test.tsx`:
  - 0 cycles → returns null
  - 1 cycle → returns null
  - 3 cycles → renders chart with correct data points
  - 10 cycles → renders chart with average reference line
- [x] Modify `ui/src/features/cycle/CyclesPage.tsx` — render `CycleLengthChart` between statistics card and current cycle section
- [x] Add chart-specific styles to `ui/src/app/theme.css` if needed
- [x] Run `make ui-lint` — must pass
- [x] Run `make ui-test` — must pass

---

### Step 5: Calendar Heatmap (VERIFIED)

Month-view grid showing colored cells for days with observations.

- [x] Create `ui/src/features/cycle/CalendarHeatmap.tsx`:
  - Custom HTML grid (no extra library), 7 columns (Sun–Sat), ~5 rows per month
  - Each cell ~40x40px with 2px gap (meets 44x44px touch target)
  - Day number in corner of each cell
  - Colored dot/fill by observation type (bleeding, symptom, mood, medication)
  - Phase color as subtle background when phase estimates are available
  - Month prev/next navigation buttons
  - Tap a day cell to show detail tooltip (observation summary)
  - Each cell gets `aria-label` describing contents (e.g., "March 15: bleeding (medium), cramps")
- [x] Create `ui/src/features/cycle/CalendarHeatmap.test.tsx`:
  - Correct number of rows/columns for a given month
  - Observation-to-color mapping is correct
  - Empty month renders with no colored cells
  - Navigation changes the displayed month
- [x] Modify `ui/src/features/cycle/CyclesPage.tsx` — render heatmap as new section with month navigation
- [x] Add heatmap styles to `ui/src/app/theme.css`:
  - `.om-heatmap-grid` — grid container
  - `.om-heatmap-cell` — day cell styling
  - `.om-heatmap-nav` — month navigation buttons
  - Dark mode variants
- [x] Data source: `ListTimeline` with month date range, grouped by date
- [x] Run `make ui-lint` — must pass
- [x] Run `make ui-test` — must pass

---

### Step 6: Mood-by-Phase Chart (VERIFIED)

Grouped bar chart showing mood type distribution across cycle phases. Answers "do I feel more anxious/sad during luteal phase?"

- [x] Create `ui/src/features/cycle/MoodPhaseChart.tsx`:
  - Recharts `BarChart` with grouped bars
  - X-axis: cycle phase (Menstruation, Follicular, Ovulation, Luteal)
  - Grouped bars per mood type, colored with `--om-color-mood` variants
  - Cross-reference mood observation timestamps against phase estimate date ranges to bucket each mood into a phase
  - Only renders when mood observations exist AND phase estimates are available
  - `role="img"` with descriptive `aria-label`
  - `.om-sr-only` text summary for screen readers
- [x] Create `ui/src/features/cycle/MoodPhaseChart.test.tsx`:
  - Correct bucketing of moods into phases
  - Empty state when no mood data
  - Empty state when no phase estimates
- [x] Data source: `ListMoodObservations` (all) + phase estimates from cycles. Map each mood's date to its phase, count occurrences per (phase, mood_type) pair
- [x] Run `make ui-lint` — must pass
- [x] Run `make ui-test` — must pass

---

### Step 7: Mood Intensity by Cycle Day Chart (VERIFIED)

Line chart plotting average mood intensity by cycle day, aggregated across multiple cycles. Answers "which days of my cycle are my worst mood days?"

- [x] Create `ui/src/features/cycle/MoodCycleDayChart.tsx`:
  - [x] Recharts `LineChart` or `ScatterChart`
  - [x] X-axis: cycle day (1–35)
  - [x] Y-axis: average mood intensity (1–3)
  - [x] One line per mood type, colored distinctly
  - [x] Compute cycle day as `(mood_date - cycle_start_date) + 1`
  - [x] Only renders with 2+ completed cycles containing mood data
  - [x] `role="img"` with descriptive `aria-label`
  - [x] `.om-sr-only` text summary for screen readers
- [x] Create `ui/src/features/cycle/MoodCycleDayChart.test.tsx`:
  - [x] Correct cycle day computation
  - [x] Averaging intensity across multiple cycles
  - [x] Empty state with insufficient data
- [x] Modify `ui/src/features/cycle/CyclesPage.tsx` — render both mood charts (Steps 6 & 7) in a new "Mood & Cycle" section below the calendar heatmap
- [x] Add mood chart styles to `ui/src/app/theme.css`:
  - [x] Per-mood-type color tokens for chart lines (reuse existing `--om-color-mood` or add variants)
  - [x] Dark mode variants
- [x] Run `make ui-lint` — must pass
- [x] Run `make ui-test` — must pass

---

## Sub-Phase 6C: Clinician Summary

### Step 8: Clinician Summary Page (VERIFIED)

Printable summary page designed for showing to a healthcare provider.

- [x] Create `ui/src/features/summary/ClinicianSummaryPage.tsx`:
  - Fetch on mount: `GetUserProfile`, `GetCycleStatistics`, `ListCycles` (last 6), `ListMedications` (active only), `ListPredictions`, `ListInsights`
  - Render structured sections:
    1. Header: "Cycle Health Summary" + generation date
    2. Profile: cycle model, regularity
    3. Statistics table: avg, median, min, max, std dev, count
    4. Recent cycles table: start date, end date, length, source
    5. Active medications list: name, category
    6. Current predictions: type, date range, confidence
    7. Insights: type, summary, confidence
  - "Print" button at top calling `window.print()`
  - No interactive elements beyond the print button
  - Semantic HTML tables with `<thead>`, `<th scope="col">`, `<caption>`
  - Clear heading hierarchy (h1 > h2 per section)
  - Handle empty sections with "No data available" messages
  - Handle loading state
- [x] Create `ui/src/features/summary/ClinicianSummaryPage.test.tsx`:
  - All sections render with mock data
  - Empty sections show "No data" messages
  - Print button is present
- [x] Add route `/summary/` → `ClinicianSummaryPage` in `ui/src/app/App.tsx`
- [x] Add "Clinician Summary" item to `ui/src/pages/SettingsPage.tsx`, linking to `/summary/`
- [x] Add styles to `ui/src/app/theme.css`:
  - [x] `.clinician-summary` scoped styles (clean typography, compact tables, clear section headers)
  - [x] `@media print` block:
    - [x] Hide navigation bar, toolbar, and print button
    - [x] Force light background (override dark mode)
    - [x] Clean typography for paper output
    - [x] Page break handling between sections
- [x] Run `make ui-lint` — must pass
- [x] Run `make ui-test` — must pass

---

### Step 9: Final Accessibility Pass (VERIFIED)

Re-audit all Phase 6 components plus any gaps found in earlier steps.

- [x] Verify all charts have `role="img"` + descriptive `aria-label`
- [x] Verify all charts have `.om-sr-only` text alternative summaries
- [x] Verify color contrast on all new design tokens (chart colors, heatmap colors, mood type colors)
- [x] Verify clinician summary tables have proper `<thead>`, `<th>`, `<caption>` elements
- [x] Verify export page is keyboard-navigable
- [x] Verify calendar heatmap cells have descriptive `aria-label` attributes
- [x] Run `make ui-lint` — must pass
- [x] Run `make ui-test` — must pass
- [x] Run `make ci` — must pass (full CI suite)

---

## Implementation Order

```
6A: Step 1 (Export) → Step 2 (Accessibility foundation)
6B: Step 3 (Recharts setup) → Step 4 (Cycle trend) → Step 5 (Calendar heatmap) → Step 6 (Mood by phase) → Step 7 (Mood by cycle day)
6C: Step 8 (Clinician summary) → Step 9 (Final a11y pass)
```

Each sub-phase is independently committable and shippable.

---

## Key Reusable Code

- `ui/src/lib/client.ts` — RPC client, all new features call existing RPCs through this
- `ui/src/lib/enums.ts` — Enum label mappings, reuse for CSV export and chart labels
- `ui/src/lib/dates.ts` — Date formatting utilities (`formatDate`, `toLocalDate`, `daysAgo`)
- `ui/src/components/EmptyState.tsx` — Reuse for empty chart/export states
- `ui/src/app/theme.css` — All design tokens, all new styles go here
