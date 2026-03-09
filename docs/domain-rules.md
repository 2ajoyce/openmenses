# Domain Rules — opencycle

This document describes the core domain rules for the opencycle application.
All rules listed here must be implemented in the Go engine (`engine/`). The UI must not re-implement them.

> **Note:** This is a living document. Rules will be added as the proto schema and engine are developed.

## Guiding Principles

1. **Offline-first.** All rules must operate without a network connection.
2. **Privacy-first.** No user data leaves the device without explicit user action.
3. **User is the authority.** The app may suggest or predict, but the user's logged data is always correct.
4. **Conservative predictions.** When uncertain, prefer conservative estimates and communicate uncertainty clearly.

## Cycle Tracking

_(Rules to be defined when proto schema is added.)_

## Symptom Logging

_(Rules to be defined when proto schema is added.)_

## Predictions

_(Rules to be defined when proto schema is added.)_

## Validation

_(Validation rules for domain objects to be defined when proto schema is added.)_
