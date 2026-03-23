import { Block, Navbar, Page } from "framework7-react";
import React from "react";

const policy = `
## Overview

OpenMenses is designed to be completely private. All of your data stays on your device — we never collect, transmit, or store any information on external servers.

## Data Collection

OpenMenses does **not** collect, share, or sell any personal data. Specifically:

- No accounts or sign-up required.
- No analytics or telemetry.
- No advertising or ad tracking.
- No network requests to external servers. The app communicates only with a local on-device engine via your device's loopback interface.
- No crash reporting services.

## Data Storage

All data you enter — including observations, cycle history, medications, and profile information — is stored in a local database on your device. This data never leaves your device unless you explicitly choose to export it.

## Data Export

You may export your data at any time from the Settings screen. Exported data is handled entirely by your device's built-in share sheet — OpenMenses has no visibility into what you do with the exported file.

## Third-Party Services

OpenMenses uses **no** third-party services, SDKs, or frameworks that collect data.

## Children's Privacy

OpenMenses does not knowingly collect information from children under 13. The app does not collect information from anyone.

## Changes to This Policy

If this policy changes, the updated version will be included in the app update. Since no data is collected, changes would only reflect new capabilities.

## Contact

OpenMenses is an open-source project. For questions about this policy, visit the project repository on GitHub.
`;

/** Minimal Markdown→HTML: headers, bold, list items, paragraphs. */
function renderMarkdown(md: string): string {
  return md
    .split("\n\n")
    .map((block) => {
      const trimmed = block.trim();
      if (!trimmed) return "";
      if (trimmed.startsWith("## ")) {
        return `<h3>${trimmed.slice(3)}</h3>`;
      }
      const lines = trimmed.split("\n");
      if (lines.every((l) => l.startsWith("- "))) {
        const items = lines
          .map((l) => `<li>${l.slice(2).replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")}</li>`)
          .join("");
        return `<ul>${items}</ul>`;
      }
      return `<p>${trimmed.replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")}</p>`;
    })
    .join("");
}

const PrivacyPolicyPage: React.FC = () => {
  return (
    <Page>
      <Navbar title="Privacy Policy" backLink="Back" />
      <Block>
        <div dangerouslySetInnerHTML={{ __html: renderMarkdown(policy) }} />
      </Block>
    </Page>
  );
};

export default PrivacyPolicyPage;
