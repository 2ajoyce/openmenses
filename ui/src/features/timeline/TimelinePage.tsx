import React, { useState, useEffect, useCallback } from "react";
import { Page, Navbar, Block, Chip } from "framework7-react";
import type { Router } from "framework7/types";
import type { TimelineRecord } from "@gen/openmenses/v1/service_pb";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { toLocalDate, daysAgo } from "../../lib/dates";
import { TimelineItem } from "./TimelineItem";
import { EmptyState } from "../../components/EmptyState";

type FilterType =
  | "bleedingObservation"
  | "symptomObservation"
  | "moodObservation"
  | "medicationEvent";

const FILTER_OPTIONS: { key: FilterType; label: string }[] = [
  { key: "bleedingObservation", label: "Bleeding" },
  { key: "symptomObservation", label: "Symptoms" },
  { key: "moodObservation", label: "Mood" },
  { key: "medicationEvent", label: "Medication" },
];

interface TimelinePageProps {
  f7router: Router.Router;
}

const TimelinePage: React.FC<TimelinePageProps> = ({ f7router }) => {
  const [records, setRecords] = useState<TimelineRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [nextPageToken, setNextPageToken] = useState("");
  const [activeFilters, setActiveFilters] = useState<Set<FilterType>>(
    new Set(),
  );

  const fetchTimeline = useCallback(async (pageToken = "") => {
    try {
      const res = await client.listTimeline({
        parent: DEFAULT_PARENT,
        range: {
          start: toLocalDate(daysAgo(30)),
          end: toLocalDate(new Date()),
        },
        pagination: { pageSize: 50, pageToken },
      });

      if (pageToken) {
        setRecords((prev) => [...prev, ...res.records]);
      } else {
        setRecords(res.records);
      }
      setNextPageToken(res.pagination?.nextPageToken ?? "");
    } catch (err) {
      console.error("Failed to fetch timeline:", err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTimeline();
  }, [fetchTimeline]);

  function handleRefresh(done: () => void) {
    fetchTimeline().then(done);
  }

  function loadMore() {
    if (nextPageToken) {
      fetchTimeline(nextPageToken);
    }
  }

  function toggleFilter(key: FilterType) {
    setActiveFilters((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  }

  const filteredRecords =
    activeFilters.size === 0
      ? records
      : records.filter((r) =>
          activeFilters.has(r.record.case as FilterType),
        );

  return (
    <Page
      ptr
      onPtrRefresh={handleRefresh}
      infinite
      infiniteDistance={50}
      onInfinite={loadMore}
      infinitePreloader={Boolean(nextPageToken)}
    >
      <Navbar title="Timeline" />

      <Block
        style={{
          display: "flex",
          flexWrap: "wrap",
          gap: "8px",
          paddingBottom: 0,
        }}
      >
        {FILTER_OPTIONS.map(({ key, label }) => (
          <Chip
            key={key}
            text={label}
            {...(activeFilters.has(key) ? { mediaBgColor: "primary" } : {})}
            outline={!activeFilters.has(key)}
            onClick={() => toggleFilter(key)}
          />
        ))}
      </Block>

      {!loading && filteredRecords.length === 0 && (
        <EmptyState
          message="No observations logged yet"
          actionLabel="Log your first observation"
          onAction={() => f7router.navigate("/log/")}
        />
      )}

      {filteredRecords.map((record, i) => (
        <TimelineItem
          key={`${record.record.case}-${i}`}
          record={record}
          onNavigateEdit={(path) => f7router.navigate(path)}
          onDeleted={() => fetchTimeline()}
        />
      ))}
    </Page>
  );
};

export default TimelinePage;
