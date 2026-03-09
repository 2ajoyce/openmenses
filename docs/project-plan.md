# Project Plan — opencycle

## Goal

Build an offline-first, privacy-first cycle tracking application that works entirely on-device with no central server.

## Phases

### Phase 1 — Foundation (current)
- [x] Initialize monorepo structure
- [ ] Define Protobuf schema (`proto/opencycle/v1/`)
- [ ] Scaffold Go engine with storage layer
- [ ] Scaffold TypeScript UI shell

### Phase 2 — Core Domain
- [ ] Implement cycle tracking domain rules in Go engine
- [ ] Implement local SQLite storage
- [ ] Wire proto-generated types through the engine
- [ ] Build basic UI screens (log entry, calendar view)

### Phase 3 — Predictions & Insights
- [ ] Implement cycle length predictions
- [ ] Implement symptom insights
- [ ] Surface predictions in UI

### Phase 4 — Mobile Wrappers
- [ ] iOS thin wrapper hosting Go engine + WebView
- [ ] Android thin wrapper hosting Go engine + WebView
- [ ] Local data export/import

### Phase 5 — Polish
- [ ] Accessibility pass
- [ ] Localization scaffolding
- [ ] Performance profiling

## Non-goals

- Central server or cloud sync
- Telemetry or analytics
- Social features
