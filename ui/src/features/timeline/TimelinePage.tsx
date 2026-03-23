import type {
  BiologicalCycleModel,
  PhaseEstimate,
} from "@gen/openmenses/v1/model_pb";
import type { TimelineRecord } from "@gen/openmenses/v1/service_pb";
import { Block, Chip, f7, Navbar, Page } from "framework7-react";
import type { Router } from "framework7/types";
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { DateTimePicker } from "../../components/DateTimePicker";
import { EmptyState } from "../../components/EmptyState";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { daysAgo, daysFromNow, toLocalDate } from "../../lib/dates";
import { TimelineItem } from "./TimelineItem";

type FilterType =
  | "bleedingObservation"
  | "symptomObservation"
  | "moodObservation"
  | "medicationEvent"
  | "cycle"
  | "phaseEstimate"
  | "prediction"
  | "insight";

interface ProcessedTimelineRecord extends TimelineRecord {
  _groupedPhaseEstimates?: PhaseEstimate[];
}

const FILTER_OPTIONS: { key: FilterType; label: string; chipClass: string }[] =
  [
    { key: "bleedingObservation", label: "Bleeding", chipClass: "bleeding" },
    { key: "symptomObservation", label: "Symptoms", chipClass: "symptom" },
    { key: "moodObservation", label: "Mood", chipClass: "mood" },
    { key: "medicationEvent", label: "Medication", chipClass: "medication" },
    { key: "cycle", label: "Cycles", chipClass: "cycle" },
    { key: "phaseEstimate", label: "Phases", chipClass: "phase" },
    { key: "prediction", label: "Predictions", chipClass: "prediction" },
    { key: "insight", label: "Insights", chipClass: "insight" },
  ];

interface TimelinePageProps {
  f7router: Router.Router;
}

const TimelinePage: React.FC<TimelinePageProps> = ({ f7router }) => {
  const [loading, setLoading] = useState(true);
  const [nextPageToken, setNextPageToken] = useState("");
  const [activeFilters, setActiveFilters] = useState<Set<FilterType>>(
    new Set(),
  );
  const [medicationNames, setMedicationNames] = useState<
    Record<string, string>
  >({});
  const [biologicalCycleModel, setBiologicalCycleModel] = useState<
    BiologicalCycleModel | undefined
  >();
  const [processedRecords, setProcessedRecords] = useState<
    ProcessedTimelineRecord[]
  >([]);
  const [startDate, setStartDate] = useState(() => daysAgo(30));
  const [endDate, setEndDate] = useState(() => daysFromNow(1));
  const loadingMoreRef = useRef(false);
  const processedRecordsRef = useRef<ProcessedTimelineRecord[]>([]);

  const recordLookup = useMemo(() => {
    const lookup: Record<string, ProcessedTimelineRecord> = {};
    for (const r of processedRecords) {
      const val = r.record.value as { name?: string } | undefined;
      if (val?.name) lookup[val.name] = r;
    }
    return lookup;
  }, [processedRecords]);

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

  const fetchUserProfile = useCallback(async () => {
    try {
      const res = await client.getUserProfile({ name: DEFAULT_PARENT });
      setBiologicalCycleModel(res.profile?.biologicalCycle);
    } catch {
      // Non-critical: biologicalCycleModel just won't be set
    }
  }, []);

  const processRecordsWithGrouping = useCallback((recs: TimelineRecord[]) => {
    // Group consecutive phase estimate records
    const processed: ProcessedTimelineRecord[] = [];
    let i = 0;

    while (i < recs.length) {
      const record = recs[i]!;

      if (record.record.case === "phaseEstimate") {
        // Collect consecutive phase estimates
        const group: PhaseEstimate[] = [record.record.value as PhaseEstimate];
        let j = i + 1;

        while (
          j < recs.length &&
          recs[j]!.record.case === "phaseEstimate" &&
          (recs[j]!.record.value as PhaseEstimate).phase ===
            (record.record.value as PhaseEstimate).phase
        ) {
          group.push(recs[j]!.record.value as PhaseEstimate);
          j++;
        }

        // Create a record with the grouped estimates
        const groupedRecord: ProcessedTimelineRecord = {
          ...record,
          _groupedPhaseEstimates: group,
        };
        processed.push(groupedRecord);
        i = j;
      } else {
        processed.push(record);
        i++;
      }
    }

    processedRecordsRef.current = processed;
    setProcessedRecords(processed);
  }, []);

  const fetchTimeline = useCallback(
    async (pageToken = "") => {
      if (!pageToken) setLoading(true);
      try {
        const res = await client.listTimeline({
          parent: DEFAULT_PARENT,
          range: {
            start: toLocalDate(startDate),
            end: toLocalDate(endDate),
          },
          pagination: { pageSize: 50, pageToken },
        });

        const newRecords = pageToken
          ? [...processedRecordsRef.current, ...res.records]
          : res.records;
        processRecordsWithGrouping(newRecords);
        setNextPageToken(res.pagination?.nextPageToken ?? "");
      } catch (err) {
        console.error("Failed to fetch timeline:", err);
      } finally {
        setLoading(false);
        loadingMoreRef.current = false;
      }
    },
    [startDate, endDate, processRecordsWithGrouping],
  );

  useEffect(() => {
    fetchTimeline();
    fetchMedicationNames();
    fetchUserProfile();
  }, [fetchTimeline, fetchMedicationNames, fetchUserProfile]);

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
    Promise.all([
      fetchTimeline(),
      fetchMedicationNames(),
      fetchUserProfile(),
    ]).then(done);
  }

  const loadMore = useCallback(() => {
    if (nextPageToken && !loadingMoreRef.current) {
      loadingMoreRef.current = true;
      fetchTimeline(nextPageToken);
    }
  }, [nextPageToken, fetchTimeline]);

  // Eagerly load all remaining pages in the background so that insight evidence
  // records (cycles, bleeding obs, etc.) are always present in recordLookup.
  // For an offline-first app with local SQLite the extra round-trips are cheap.
  useEffect(() => {
    if (!nextPageToken || loadingMoreRef.current) return;
    loadingMoreRef.current = true;
    fetchTimeline(nextPageToken);
  }, [nextPageToken, fetchTimeline]);

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
      ? processedRecords
      : processedRecords.filter((r) => {
          const c = r.record.case;
          if (activeFilters.has(c as FilterType)) return true;
          // "Medication" chip also shows medication resource records
          if (c === "medication" && activeFilters.has("medicationEvent"))
            return true;
          return false;
        });

  return (
    <Page
      pageContent={false}
      ptr
      onPtrRefresh={handleRefresh}
      infinite
      infiniteDistance={50}
      onInfinite={loadMore}
      infinitePreloader={Boolean(nextPageToken)}
      onPageBeforeIn={() => fetchTimeline()}
    >
      <div className="page-content ptr-content infinite-scroll-content">
        <Navbar title="Timeline" />

        <Block className="timeline-control-row">
          <DateTimePicker
            label="From"
            value={startDate}
            onChange={setStartDate}
          />
          <DateTimePicker label="To" value={endDate} onChange={setEndDate} />
        </Block>

        <div role="region" aria-label="Filter timeline by observation type">
          <Block className="timeline-control-row">
            {FILTER_OPTIONS.map(({ key, label, chipClass }) => (
              <div
                key={key}
                role="button"
                tabIndex={0}
                aria-pressed={activeFilters.has(key)}
                aria-label={label}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    toggleFilter(key);
                  }
                }}
              >
                <Chip
                  text={label}
                  className={`om-chip-${chipClass}${activeFilters.has(key) ? " om-chip-active" : ""}`}
                  onClick={() => toggleFilter(key)}
                />
              </div>
            ))}
          </Block>
        </div>

        {loading && (
          <div aria-live="polite" aria-label="Loading timeline">
            <p>Loading timeline...</p>
          </div>
        )}

        {!loading && filteredRecords.length === 0 && (
          <EmptyState
            message="No observations logged yet"
            actionLabel="Log your first observation"
            onAction={() => f7router.navigate("/log/")}
          />
        )}

        <div
          role="feed"
          aria-label="Timeline observations"
          aria-live="polite"
          aria-busy={loading}
        >
          {filteredRecords.map((record) => (
            <TimelineItem
              key={`${record.record.case}-${record.record.value?.name}`}
              record={record}
              medicationNames={medicationNames}
              recordLookup={recordLookup}
              {...(biologicalCycleModel != null && { biologicalCycleModel })}
              {...(record._groupedPhaseEstimates != null && {
                groupedPhaseEstimates: record._groupedPhaseEstimates,
              })}
              onNavigateEdit={(path) => f7router.navigate(path)}
              onDeleted={() => fetchTimeline()}
            />
          ))}
        </div>
      </div>
    </Page>
  );
};

export default TimelinePage;
