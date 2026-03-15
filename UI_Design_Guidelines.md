# UI Design Guidelines

This document defines the styling and CSS conventions for the OpenMenses UI layer.

The goals are:

- visual consistency across all screens
- minimal custom CSS that ages well
- leverage Framework7's adaptive theming instead of fighting it
- clear rules so contributors (human and AI) produce uniform output

---

# 1. Core Philosophy

The UI should feel like a well-made native app, not a website.

Principles:

1. **Framework7 first** — use Framework7 components and their built-in styling before writing custom CSS.
2. **CSS custom properties for design tokens** — colors, spacing, and typography values live in a single file as CSS variables.
3. **One stylesheet** — all custom CSS lives in one file (`ui/src/app/theme.css`). No per-component CSS files, no CSS modules, no CSS-in-JS.
4. **Utility classes over inline styles** — define small reusable classes for common patterns (flex row, text truncation, color indicators). Avoid `style={{}}` in JSX.
5. **Lean on the platform** — Framework7's `theme: "auto"` adapts to iOS and Material. Do not override platform-specific styling unless there is a specific design reason.

---

# 2. File Structure

All custom styling lives in two places:

```
ui/src/app/
  theme.css       ← design tokens + utility classes + component overrides
  App.tsx          ← imports theme.css
```

No other CSS files should be created unless a future phase introduces a genuinely separate concern (e.g., a print stylesheet).

---

# 3. Design Tokens

Design tokens are CSS custom properties defined on `:root` in `theme.css`. They provide the single source of truth for visual constants.

### 3.1 Token categories

| Category        | Prefix         | Examples                                                                                |
| --------------- | -------------- | --------------------------------------------------------------------------------------- |
| Brand colors    | `--om-color-`  | `--om-color-primary`, `--om-color-primary-light`                                        |
| Semantic colors | `--om-color-`  | `--om-color-bleeding`, `--om-color-mood`, `--om-color-symptom`, `--om-color-medication` |
| Status colors   | `--om-color-`  | `--om-color-success`, `--om-color-warning`, `--om-color-error`                          |
| Spacing         | `--om-space-`  | `--om-space-xs`, `--om-space-sm`, `--om-space-md`, `--om-space-lg`                      |
| Typography      | `--om-font-`   | `--om-font-size-sm`, `--om-font-size-base`                                              |
| Radii           | `--om-radius-` | `--om-radius-sm`, `--om-radius-pill`                                                    |

### 3.2 Naming rules

- Prefix all custom properties with `--om-` to avoid collisions with Framework7's `--f7-` namespace.
- Use lowercase kebab-case.
- Use semantic names (`--om-color-bleeding`) not raw values (`--om-color-pink-3`).

### 3.3 Framework7 overrides

To change Framework7's built-in theme colors, override its CSS custom properties inside `:root`:

```css
:root {
  --f7-theme-color: var(--om-color-primary);
  --f7-theme-color-rgb: /* RGB triplet */;
}
```

Override Framework7 variables sparingly. Only change values that affect brand identity (primary color, font family). Let Framework7 handle everything else.

---

# 4. Color System

### 4.1 Observation-type colors

Each observation type has a designated color used for indicators, card accents, and filter chips:

| Type       | Token                   | Purpose               |
| ---------- | ----------------------- | --------------------- |
| Bleeding   | `--om-color-bleeding`   | Card dot, filter chip |
| Symptom    | `--om-color-symptom`    | Card dot, filter chip |
| Mood       | `--om-color-mood`       | Card dot, filter chip |
| Medication | `--om-color-medication` | Card dot, filter chip |

Sub-variants (e.g., bleeding flow levels from spotting to heavy) are defined as additional tokens: `--om-color-bleeding-spotting`, `--om-color-bleeding-light`, `--om-color-bleeding-medium`, `--om-color-bleeding-heavy`.

### 4.2 Dark mode

Framework7 handles dark mode automatically when `theme: "auto"` is set. Custom tokens must also respond to dark mode:

```css
:root {
  --om-color-card-bg: #ffffff;
}

:root .dark,
:root.dark {
  --om-color-card-bg: #1c1c1e;
}
```

Every custom color token must have a dark-mode counterpart defined in the dark block. Use Framework7's dark mode selectors (`.dark` class) for consistency with the framework's own approach.

---

# 5. Spacing

Use a fixed spacing scale. Do not invent arbitrary pixel values.

| Token            | Value  |
| ---------------- | ------ |
| `--om-space-xs`  | `4px`  |
| `--om-space-sm`  | `8px`  |
| `--om-space-md`  | `16px` |
| `--om-space-lg`  | `24px` |
| `--om-space-xl`  | `32px` |
| `--om-space-2xl` | `48px` |

When a spacing value is needed, use the nearest token. If none fits, prefer rounding to the nearest step over introducing a one-off value.

---

# 6. Utility Classes

Define small, single-purpose classes in `theme.css` for patterns that repeat across components. These replace inline `style={{}}` blocks.

### 6.1 Approved utility patterns

| Class             | Purpose                                                               |
| ----------------- | --------------------------------------------------------------------- |
| `.om-row`         | `display: flex; align-items: center; gap: var(--om-space-sm)`         |
| `.om-row-between` | `.om-row` + `justify-content: space-between`                          |
| `.om-col`         | `display: flex; flex-direction: column; gap: var(--om-space-xs)`      |
| `.om-truncate`    | Single-line text truncation (ellipsis)                                |
| `.om-truncate-2`  | Two-line clamp with ellipsis                                          |
| `.om-dot`         | Colored indicator dot (width/height from token, `border-radius: 50%`) |
| `.om-muted`       | Reduced opacity text                                                  |

### 6.2 Rules for utility classes

- Each class does one thing.
- Prefix all custom classes with `om-` to distinguish from Framework7 classes.
- Do not create utilities speculatively. Add one only when the same inline style appears in three or more components.
- Keep the total count low. If the utility list exceeds ~20 classes, reconsider whether some belong as component-level rules instead.

---

# 7. Component-Level Styles

When a style applies to a single component and is not reusable, define it as a class in `theme.css` scoped by a component class name.

```css
/* Bleeding card flow indicator colors */
.bleeding-card .om-dot[data-flow="1"] {
  background-color: var(--om-color-bleeding-spotting);
}
.bleeding-card .om-dot[data-flow="2"] {
  background-color: var(--om-color-bleeding-light);
}
.bleeding-card .om-dot[data-flow="3"] {
  background-color: var(--om-color-bleeding-medium);
}
.bleeding-card .om-dot[data-flow="4"] {
  background-color: var(--om-color-bleeding-heavy);
}
```

Rules:

- Use a descriptive top-level class on the component's root element (e.g., `bleeding-card`, `timeline-filters`).
- Keep component styles in the same `theme.css` file, grouped under a comment header for that component.
- Do not use CSS nesting deeper than two levels.

---

# 8. What Not To Do

### 8.1 No inline styles

Avoid `style={{}}` in JSX. Move all visual styling to `theme.css` using tokens and classes.

Allowed exceptions:

- Truly dynamic values computed at runtime (e.g., a progress bar width from a percentage).
- One-off prototyping during development, converted to classes before merging.

### 8.2 No CSS-in-JS

Do not introduce styled-components, Emotion, or similar libraries. The project uses plain CSS and Framework7.

### 8.3 No CSS modules

CSS modules add build complexity and fragment styles across files. Use the single `theme.css` file with `om-` prefixed class names to avoid collisions.

### 8.4 No Tailwind

Tailwind conflicts with Framework7's own class-based styling system and adds a build step. The utility class approach in this project is intentionally smaller and hand-maintained.

### 8.5 No `!important`

If a Framework7 style needs to be overridden, use a more specific selector. If that is not possible, reconsider whether the override is appropriate.

### 8.6 No magic numbers

Every spacing, font size, and color value should reference a design token. Hard-coded pixel values and hex colors in CSS rules indicate a missing token.

---

# 9. Typography

Defer to Framework7's default font stack, which uses the system font on each platform (San Francisco on iOS, Roboto on Android).

Custom typography tokens:

| Token                     | Value  | Usage                          |
| ------------------------- | ------ | ------------------------------ |
| `--om-font-size-xs`       | `11px` | Timestamps, metadata           |
| `--om-font-size-sm`       | `13px` | Secondary text, labels         |
| `--om-font-size-base`     | `16px` | Body text (Framework7 default) |
| `--om-font-size-lg`       | `20px` | Section headers                |
| `--om-font-weight-normal` | `400`  | Body text                      |
| `--om-font-weight-medium` | `500`  | Emphasis, card titles          |
| `--om-font-weight-bold`   | `600`  | Page titles                    |

Do not import custom web fonts. System fonts keep the bundle small and feel native.

---

# 10. Card Design

Timeline cards are the primary visual element. They should have clear visual hierarchy:

1. **Type indicator** — colored dot or icon identifying the observation type.
2. **Title line** — observation type and primary value (e.g., "Bleeding — Light").
3. **Timestamp** — secondary text below the title.
4. **Notes** — tertiary text, truncated to two lines.
5. **Actions** — Edit and Delete links, right-aligned in the header.

Cards should use subtle differentiation — a left border accent color or a small colored dot — rather than fully colored backgrounds, to keep the timeline scannable.

---

# 11. Responsive Considerations

The app targets mobile WebViews. It will never run on a desktop browser in production.

Rules:

- Design for 320px–428px viewport width (iPhone SE through iPhone Pro Max).
- Do not add desktop breakpoints or responsive grid systems.
- Use Framework7's `safe-areas` class for notch/status bar handling.
- Set the viewport meta tag to prevent user scaling (already configured in `index.html`).

---

# 12. Accessibility

- Use sufficient color contrast (WCAG AA minimum: 4.5:1 for text, 3:1 for large text).
- Do not rely on color alone to convey meaning — pair color indicators with text labels.
- Use semantic HTML elements and Framework7's built-in ARIA support.
- Ensure touch targets are at least 44x44px (Framework7 components handle this by default).

---

# 13. Icons

Use Framework7's built-in icon set (`framework7-icons`). Do not add additional icon libraries unless the built-in set lacks a required icon.

If a custom icon is needed, add it as an inline SVG component, not as a new font or icon library dependency.

---

# 14. Checklist for New Components

Before merging a new UI component, verify:

- [ ] All colors reference `--om-` tokens or Framework7 variables.
- [ ] All spacing uses `--om-space-` tokens.
- [ ] No inline `style={{}}` blocks (unless truly dynamic).
- [ ] Custom classes use `om-` prefix.
- [ ] Dark mode appearance tested (toggle device/browser dark mode).
- [ ] Component is readable on a 320px-wide viewport.
- [ ] Touch targets are at least 44x44px.

---

# 15. Guidance for AI Agents

Agents generating UI code should:

1. Import and use classes from `theme.css` instead of writing inline styles.
2. Reference `--om-` tokens for all color, spacing, and typography values.
3. Use Framework7 components (Button, Card, List, Block, etc.) for structure.
4. Add new utility classes to `theme.css` only when the same pattern appears in three or more places.
5. Never introduce new CSS files, CSS-in-JS libraries, or build-time CSS tools.
6. Test that components render correctly in both light and dark mode.
7. Keep selectors flat — two levels of nesting maximum.
