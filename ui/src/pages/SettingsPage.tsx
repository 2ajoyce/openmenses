import React from "react";
import { Page, Navbar, Block, BlockTitle } from "framework7-react";

const SettingsPage: React.FC = () => {
  return (
    <Page>
      <Navbar title="Settings" />
      <BlockTitle>User Profile</BlockTitle>
      <Block>
        <p>Settings and profile management coming soon.</p>
      </Block>
    </Page>
  );
};

export default SettingsPage;
