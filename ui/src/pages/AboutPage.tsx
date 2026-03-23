import { Block, BlockTitle, Navbar, Page } from "framework7-react";
import React from "react";

const AboutPage: React.FC = () => {
  return (
    <Page>
      <Navbar title="About OpenMenses" backLink="Back" />
      <Block>
        <p>
          <strong>OpenMenses</strong> is an open-source, offline-first,
          privacy-first menstrual and cycle tracking application.
        </p>
        <p>
          All data stays on your device. There is no central server, no
          telemetry, and no cloud services.
        </p>
      </Block>

      <BlockTitle>Principles</BlockTitle>
      <Block>
        <ul>
          <li>
            <strong>Offline-first</strong> - works without an internet
            connection.
          </li>
          <li>
            <strong>User-owned data</strong> - your data never leaves your
            device unless you export it.
          </li>
          <li>
            <strong>Transparent algorithms</strong> - cycle detection,
            predictions, and insights are fully open-source.
          </li>
          <li>
            <strong>No assumptions</strong> - the app avoids assumptions about
            identity or reproductive goals.
          </li>
          <li>
            <strong>Open data model</strong> - the data schema is defined in
            Protocol Buffers and is publicly documented.
          </li>
        </ul>
      </Block>

      <BlockTitle>How It Works</BlockTitle>
      <Block>
        <p>
          OpenMenses uses a local Go engine running on-device to handle all
          domain logic - cycle detection, phase estimation, predictions, and
          data validation. The user interface is a TypeScript layer that
          communicates with the engine entirely on your device.
        </p>
        <p>
          You can track bleeding, symptoms, moods, and medications. The engine
          derives cycles, estimates phases, and generates predictions and
          insights based on your data.
        </p>
      </Block>

      <BlockTitle>Open Source</BlockTitle>
      <Block>
        <p>
          OpenMenses is free and open-source software. The source code,
          documentation, and domain rules are publicly available on GitHub.
        </p>
      </Block>

      <Block>
        <p style={{ textAlign: "center", opacity: 0.5 }}>
          Version 1.0 &middot; Build 1
        </p>
      </Block>
    </Page>
  );
};

export default AboutPage;
