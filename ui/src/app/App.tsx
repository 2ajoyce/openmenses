import {
  App as F7App,
  Link,
  Toolbar,
  ToolbarPane,
  View,
  Views,
} from "framework7-react";
import type { Framework7Parameters } from "framework7/types";
import React from "react";

import BleedingForm from "../features/bleeding/BleedingForm";
import CyclesPage from "../features/cycle/CyclesPage";
import ExportPage from "../features/export/ExportPage";
import MedicationEventForm from "../features/medication/MedicationEventForm";
import MedicationForm from "../features/medication/MedicationForm";
import MedicationList from "../features/medication/MedicationList";
import MoodForm from "../features/mood/MoodForm";
import ClinicianSummaryPage from "../features/summary/ClinicianSummaryPage";
import SymptomForm from "../features/symptom/SymptomForm";
import TimelinePage from "../features/timeline/TimelinePage";
import AboutPage from "../pages/AboutPage";
import LogChooserPage from "../pages/LogChooserPage";
import PrivacyPolicyPage from "../pages/PrivacyPolicyPage";
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
  { path: "/cycles/", component: CyclesPage },
  { path: "/export/", component: ExportPage },
  { path: "/summary/", component: ClinicianSummaryPage },
  { path: "/settings/", component: SettingsPage },
  { path: "/privacy/", component: PrivacyPolicyPage },
  { path: "/about/", component: AboutPage },
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
        <Toolbar tabbar icons bottom>
          <ToolbarPane>
            <Link
              tabLink="#tab-timeline"
              tabLinkActive
              text="Timeline"
              iconF7="clock"
            />
            <Link tabLink="#tab-cycles" text="Cycles" iconF7="circle_grid_hex" />
            <Link tabLink="#tab-log" text="Log" iconF7="plus_circle" />
            <Link
              tabLink="#tab-medications"
              text="Medications"
              iconF7="capsule"
            />
            <Link tabLink="#tab-settings" text="Settings" iconF7="gear" />
          </ToolbarPane>
        </Toolbar>

        <View
          id="tab-timeline"
          tab
          tabActive
          url="/"
          main
        />
        <View id="tab-cycles" tab url="/cycles/" />
        <View id="tab-log" tab url="/log/" />
        <View id="tab-medications" tab url="/medications/" />
        <View id="tab-settings" tab url="/settings/" />
      </Views>
    </F7App>
  );
};

export default App;
