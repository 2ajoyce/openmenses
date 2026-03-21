import React from "react";
import {
  Page,
  Navbar,
  List,
  ListItem,
  BlockTitle,
  Icon,
} from "framework7-react";

const LogChooserPage: React.FC = () => {
  return (
    <Page pageContent={false}>
      <div className="page-content">
      <Navbar title="Log Observation" />
      <BlockTitle>What would you like to log?</BlockTitle>
      <List inset>
        <ListItem
          title="Bleeding"
          subtitle="Track flow and timing"
          link="/log/bleeding/"
          mediaItem
        >
          <Icon slot="media" f7="drop_fill" className="log-chooser-icon-bleeding" />
        </ListItem>
        <ListItem
          title="Symptom"
          subtitle="Log cramps, headaches and more"
          link="/log/symptom/"
          mediaItem
        >
          <Icon slot="media" f7="bandage" className="log-chooser-icon-symptom" />
        </ListItem>
        <ListItem
          title="Mood"
          subtitle="Record your emotional state"
          link="/log/mood/"
          mediaItem
        >
          <Icon slot="media" f7="face_smiling" className="log-chooser-icon-mood" />
        </ListItem>
        <ListItem
          title="Medication"
          subtitle="Log taken, missed or skipped doses"
          link="/log/medication/"
          mediaItem
        >
          <Icon slot="media" f7="capsule" className="log-chooser-icon-medication" />
        </ListItem>
      </List>
      </div>
    </Page>
  );
};

export default LogChooserPage;
