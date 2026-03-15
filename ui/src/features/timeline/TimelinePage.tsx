import type { TimelineRecord } from "@gen/openmenses/v1/service_pb";
import { Block, Chip, f7, Navbar, Page } from "framework7-react";
import type { Router } from "framework7/types";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { DateTimePicker } from "../../components/DateTimePicker";
import { EmptyState } from "../../components/EmptyState";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { daysAgo, toLocalDate } from "../../lib/dates";
import { TimelineItem } from "./TimelineItem";

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
  const [medicationNames, setMedicationNames] = useState<
    Record<string, string>
  >({});
  const [startDate, setStartDate] = useState(() => daysAgo(30));
  const [endDate, setEndDate] = useState(() => new Date());
  const loadingMoreRef = useRef(false);

  const fetchMedicationNames = useCallback(async () => {
    try {
      const res = await client.listMedications({
        parent: DEFAULT_PARENT,
        pagination: { pageSize: 100, pageToken: "" },
      });
      const lookup: Record<string, string> = {};
      for (const med of res.medications) {
        lookup[med.name] = med.displayName;
      }
      setMedicationNames(lookup);
    } catch {
      // Non-critical: cards fall back to "Medication"
    }
  }, []);

  const fetchTimeline = useCallback(
    async (pageToken = "") => {
      try {
        const res = await client.listTimeline({
          parent: DEFAULT_PARENT,
          range: {
            start: toLocalDate(startDate),
            end: toLocalDate(endDate),
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
        loadingMoreRef.current = false;
      }
    },
    [startDate, endDate],
  );

  useEffect(() => {
    fetchTimeline();
    fetchMedicationNames();
  }, [fetchTimeline, fetchMedicationNames]);

  useEffect(() => {
    const handleTabShow = (tabEl: HTMLElement) => {
      if (tabEl.id === "tab-timeline") {
        fetchTimeline();
      }
    };
    f7.on("tabShow", handleTabShow);
    return () => {
      f7.off("tabShow", handleTabShow);
    };
  }, [fetchTimeline]);

  function handleRefresh(done: () => void) {
    Promise.all([fetchTimeline(), fetchMedicationNames()]).then(done);
  }

  function loadMore() {
    if (nextPageToken && !loadingMoreRef.current) {
      loadingMoreRef.current = true;
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
      : records.filter((r) => {
          const c = r.record.case;
          if (activeFilters.has(c as FilterType)) return true;
          // "Medication" chip also shows medication resource records
          if (c === "medication" && activeFilters.has("medicationEvent"))
            return true;
          return false;
        });

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
        <DateTimePicker
          label="From"
          value={startDate}
          onChange={setStartDate}
        />
        <DateTimePicker label="To" value={endDate} onChange={setEndDate} />
      </Block>

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

      {filteredRecords.map((record) => (
        <TimelineItem
          key={`${record.record.case}-${record.record.value?.name}`}
          record={record}
          medicationNames={medicationNames}
          onNavigateEdit={(path) => f7router.navigate(path)}
          onDeleted={() => fetchTimeline()}
        />
      ))}
    </Page>
  );
};

export default TimelinePage;
