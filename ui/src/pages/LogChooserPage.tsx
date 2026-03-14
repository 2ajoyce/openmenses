import React from "react";
import {
  Page,
  Navbar,
  List,
  ListItem,
  BlockTitle,
} from "framework7-react";

const LogChooserPage: React.FC = () => {
  return (
    <Page>
      <Navbar title="Log Observation" />
      <BlockTitle>What would you like to log?</BlockTitle>
      <List inset>
        <ListItem
          title="Bleeding"
          link="/log/bleeding/"
        />
        <ListItem
          title="Symptom"
          link="/log/symptom/"
        />
        <ListItem
          title="Mood"
          link="/log/mood/"
        />
        <ListItem
          title="Medication"
          link="/log/medication/"
        />
      </List>
    </Page>
  );
};

export default LogChooserPage;
