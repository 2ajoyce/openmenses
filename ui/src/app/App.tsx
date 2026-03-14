import React from "react";
import {
  App as F7App,
  View,
  Views,
  Toolbar,
  Link,
} from "framework7-react";
import type { Framework7Parameters } from "framework7/types";

import TimelinePage from "../features/timeline/TimelinePage";
import LogChooserPage from "../pages/LogChooserPage";
import BleedingForm from "../features/bleeding/BleedingForm";
import SymptomForm from "../features/symptom/SymptomForm";
import MoodForm from "../features/mood/MoodForm";
import MedicationEventForm from "../features/medication/MedicationEventForm";
import MedicationList from "../features/medication/MedicationList";
import MedicationForm from "../features/medication/MedicationForm";
import SettingsPage from "../pages/SettingsPage";

const routes: Framework7Parameters["routes"] = [
  { path: "/", component: TimelinePage },
  { path: "/log/", component: LogChooserPage },
  { path: "/log/bleeding/", component: BleedingForm },
  { path: "/log/symptom/", component: SymptomForm },
  { path: "/log/mood/", component: MoodForm },
  { path: "/log/medication/", component: MedicationEventForm },
  { path: "/medications/", component: MedicationList },
  { path: "/medications/new/", component: MedicationForm },
  { path: "/medications/edit/", component: MedicationForm },
  { path: "/settings/", component: SettingsPage },
];

const f7params: Framework7Parameters = {
  name: "OpenMenses",
  theme: "auto",
  routes,
};

const App: React.FC = () => {
  return (
    <F7App {...f7params}>
      <Views tabs className="safe-areas">
        <Toolbar tabbar bottom>
          <Link
            tabLink="#tab-timeline"
            tabLinkActive
            text="Timeline"
            iconF7="clock"
          />
          <Link tabLink="#tab-log" text="Log" iconF7="plus_circle" />
          <Link
            tabLink="#tab-medications"
            text="Medications"
            iconF7="capsule"
          />
          <Link tabLink="#tab-settings" text="Settings" iconF7="gear" />
        </Toolbar>

        <View
          id="tab-timeline"
          tab
          tabActive
          url="/"
          main
        />
        <View id="tab-log" tab url="/log/" />
        <View id="tab-medications" tab url="/medications/" />
        <View id="tab-settings" tab url="/settings/" />
      </Views>
    </F7App>
  );
};

export default App;
