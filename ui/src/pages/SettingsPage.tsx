import React from "react";
import {
  Page,
  Navbar,
  Block,
  BlockTitle,
  List,
  ListItem,
} from "framework7-react";

const SettingsPage: React.FC = () => {
  return (
    <Page>
      <Navbar title="Settings" />

      <Block className="text-align-center">
        <p className="settings-app-name">OpenMenses</p>
        <p className="om-muted">Version 0.1.0</p>
      </Block>

      <BlockTitle>Data</BlockTitle>
      <List inset>
        <ListItem title="Export Data" link="#" />
      </List>

      <BlockTitle>About</BlockTitle>
      <List inset>
        <ListItem title="Privacy Policy" link="#" />
        <ListItem title="About OpenMenses" link="#" />
      </List>
    </Page>
  );
};

export default SettingsPage;
