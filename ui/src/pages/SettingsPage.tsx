import React, { useState, useEffect } from "react";
import {
  Page,
  Navbar,
  Block,
  BlockTitle,
  List,
  ListItem,
  Button,
  Icon,
} from "framework7-react";
import { client, DEFAULT_PARENT } from "../lib/client";
import {
  biologicalCycleModelOptions,
  cycleRegularityOptions,
  trackingFocusOptions,
} from "../lib/enums";
import type { UserProfile } from "@gen/openmenses/v1/model_pb";
import { TrackingFocus } from "@gen/openmenses/v1/model_pb";

const SettingsPage: React.FC = () => {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showSuccess, setShowSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [biologicalCycle, setBiologicalCycle] = useState(0);
  const [cycleRegularity, setCycleRegularity] = useState(0);
  const [trackingFocuses, setTrackingFocuses] = useState<TrackingFocus[]>([]);

  useEffect(() => {
    fetchProfile();
  }, []);

  const fetchProfile = async () => {
    try {
      setLoading(true);
      const response = await client.getUserProfile({
        name: DEFAULT_PARENT,
      });
      if (response.profile) {
        setProfile(response.profile);
        setBiologicalCycle(response.profile.biologicalCycle);
        setCycleRegularity(response.profile.cycleRegularity);
        setTrackingFocuses(response.profile.trackingFocus);
      } else {
        setProfile(null);
        setBiologicalCycle(0);
        setCycleRegularity(0);
        setTrackingFocuses([]);
      }
    } catch {
      // Profile doesn't exist for first-time user, reset to defaults
      setProfile(null);
      setBiologicalCycle(0);
      setCycleRegularity(0);
      setTrackingFocuses([]);
    } finally {
      setLoading(false);
    }
  };

  const handleTrackingFocusChange = (focus: TrackingFocus, checked: boolean) => {
    if (checked) {
      setTrackingFocuses((prev) => [...prev, focus]);
    } else {
      setTrackingFocuses((prev) => prev.filter((f) => f !== focus));
    }
  };

  const handleSave = async () => {
    if (!biologicalCycle || !cycleRegularity || trackingFocuses.length === 0) {
      setError(
        "Please fill in all fields: biological cycle model, cycle regularity, and at least one tracking focus."
      );
      return;
    }

    try {
      setSaving(true);
      setError(null);

      const profileData = {
        name: DEFAULT_PARENT,
        biologicalCycle,
        cycleRegularity,
        trackingFocus: trackingFocuses,
      };

      if (profile) {
        // Update existing profile
        const response = await client.updateUserProfile({
          profile: profileData,
          updateMask: {
            paths: ["biological_cycle", "cycle_regularity", "tracking_focus"],
          },
        });
        if (response.profile) {
          setProfile(response.profile);
        }
      } else {
        // Create new profile
        const response = await client.createUserProfile({
          profile: profileData,
        });
        if (response.profile) {
          setProfile(response.profile);
        }
      }

      setShowSuccess(true);
      setTimeout(() => setShowSuccess(false), 3000);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to save profile"
      );
    } finally {
      setSaving(false);
    }
  };

  return (
    <Page>
      <Navbar title="Settings" />

      <Block className="text-align-center">
        <p className="settings-app-name">OpenMenses</p>
        <p className="om-muted">Version 0.1.0</p>
      </Block>

      {loading ? (
        <Block className="text-align-center">
          <p className="om-muted">Loading profile...</p>
        </Block>
      ) : (
        <>
          <BlockTitle>Profile</BlockTitle>
          <Block inset>
            <div className="profile-form">
              <div className="form-group">
                <label htmlFor="biological-cycle">Biological Cycle Model</label>
                <select
                  id="biological-cycle"
                  value={biologicalCycle}
                  onChange={(e) => setBiologicalCycle(Number(e.target.value))}
                  disabled={saving}
                >
                  <option value={0}>Select...</option>
                  {biologicalCycleModelOptions.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </select>
              </div>

              <div className="form-group">
                <label htmlFor="cycle-regularity">Cycle Regularity</label>
                <select
                  id="cycle-regularity"
                  value={cycleRegularity}
                  onChange={(e) => setCycleRegularity(Number(e.target.value))}
                  disabled={saving}
                >
                  <option value={0}>Select...</option>
                  {cycleRegularityOptions.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </select>
              </div>

              <div className="form-group">
                <label>Tracking Focus (select at least one)</label>
                <div className="tracking-focus-options">
                  {trackingFocusOptions.map((opt) => (
                    <label key={opt.value} className="checkbox-label">
                      <input
                        type="checkbox"
                        checked={trackingFocuses.includes(opt.value)}
                        onChange={(e) =>
                          handleTrackingFocusChange(
                            opt.value,
                            e.target.checked
                          )
                        }
                        disabled={saving}
                      />
                      <span>{opt.label}</span>
                    </label>
                  ))}
                </div>
              </div>

              {error && <div className="form-error">{error}</div>}

              {showSuccess && (
                <div className="form-success">
                  <Icon ios="f7:checkmark_circle_fill" md="material:check_circle" />
                  Profile saved successfully
                </div>
              )}

              <Button
                onClick={handleSave}
                disabled={saving}
                fill
                large
                className="save-button"
              >
                {saving ? "Saving..." : "Save Profile"}
              </Button>
            </div>
          </Block>
        </>
      )}

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
