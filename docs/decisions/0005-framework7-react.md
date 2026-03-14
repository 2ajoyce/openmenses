# ADR 0005 — Framework7 React

**Date:** 2026-03-13
**Status:** Accepted

## Context

The UI layer runs as a React application inside a WebView on both iOS and Android. A component library is needed that provides native-feeling UI components with adaptive styling (iOS style on iOS, Material style on Android) out of the box, without requiring custom theming or platform detection.

## Decision

Use Framework7 React as the UI component library.

## Consequences

**Positive:**
- Adaptive styling: Framework7 automatically renders iOS-style components on iOS and Material-style components on Android, matching platform conventions without manual theming.
- Built-in mobile navigation patterns (stack navigation, tab bars, action sheets, pull-to-refresh) that are standard in native mobile apps.
- Designed specifically for mobile web apps in WebViews, aligning with the project's architecture.
- Includes touch-optimized interactions (swipe gestures, haptic-style feedback) that improve the user experience on mobile devices.

**Negative:**
- Smaller community compared to libraries like MUI or Ionic, which may mean fewer third-party plugins and community resources.
- Tighter coupling to Framework7's component API, making a future migration to a different library more involved.

## Alternatives Considered

- **Ionic React:** Mature and widely used, but adds its own runtime layer and Capacitor dependency, which conflicts with the project's approach of using a thin native shell with a Go engine.
- **shadcn/ui + Tailwind CSS:** Highly customizable and popular in the React ecosystem, but designed for desktop-first web apps. Does not provide adaptive iOS/Android styling or mobile navigation patterns out of the box.
- **Radix UI:** Excellent accessibility primitives, but unstyled by design. Would require significant custom work to achieve native-feeling mobile UI on both platforms.
- **MUI (Material UI):** Large and well-maintained, but strongly opinionated toward Material Design. Does not provide adaptive iOS styling, so the app would look non-native on iOS devices.
