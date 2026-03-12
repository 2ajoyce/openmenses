// Package memory provides a thread-safe in-memory implementation of
// the storage.Repository interface. It is intended for use in tests and
// as the storage backend for the engine-dev CLI when no SQLite path is given.
package memory

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

const defaultPageSize = 50

// Store is the root in-memory repository.
type Store struct {
	profiles         *userProfileStore
	bleedings        *bleedingStore
	symptoms         *symptomStore
	moods            *moodStore
	medications      *medicationStore
	medicationEvents *medicationEventStore
	cycles           *cycleStore
	phaseEstimates   *phaseEstimateStore
	predictions      *predictionStore
	insights         *insightStore
}

// New returns an empty in-memory Store.
func New() *Store {
	return &Store{
		profiles:         &userProfileStore{data: map[string]*v1.UserProfile{}},
		bleedings:        &bleedingStore{data: map[string]*v1.BleedingObservation{}},
		symptoms:         &symptomStore{data: map[string]*v1.SymptomObservation{}},
		moods:            &moodStore{data: map[string]*v1.MoodObservation{}},
		medications:      &medicationStore{data: map[string]*v1.Medication{}},
		medicationEvents: &medicationEventStore{data: map[string]*v1.MedicationEvent{}},
		cycles:           &cycleStore{data: map[string]*v1.Cycle{}},
		phaseEstimates:   &phaseEstimateStore{data: map[string]*v1.PhaseEstimate{}},
		predictions:      &predictionStore{data: map[string]*v1.Prediction{}},
		insights:         &insightStore{data: map[string]*v1.Insight{}},
	}
}

// Verify Store implements storage.Repository at compile time.
var _ storage.Repository = (*Store)(nil)

func (s *Store) UserProfiles() storage.UserProfileRepository { return s.profiles }
func (s *Store) BleedingObservations() storage.BleedingObservationRepository {
	return s.bleedings
}
func (s *Store) SymptomObservations() storage.SymptomObservationRepository { return s.symptoms }
func (s *Store) MoodObservations() storage.MoodObservationRepository       { return s.moods }
func (s *Store) Medications() storage.MedicationRepository                 { return s.medications }
func (s *Store) MedicationEvents() storage.MedicationEventRepository {
	return s.medicationEvents
}
func (s *Store) Cycles() storage.CycleRepository                 { return s.cycles }
func (s *Store) PhaseEstimates() storage.PhaseEstimateRepository { return s.phaseEstimates }
func (s *Store) Predictions() storage.PredictionRepository       { return s.predictions }
func (s *Store) Insights() storage.InsightRepository             { return s.insights }

// ---- helpers ----------------------------------------------------------------

func pageItems[T proto.Message](items []T, req storage.PageRequest) (storage.ListPage[T], error) {
	size := int(req.PageSize)
	if size <= 0 {
		size = defaultPageSize
	}
	if size > 500 {
		size = 500
	}

	offset := 0
	if req.PageToken != "" {
		n, err := strconv.Atoi(req.PageToken)
		if err != nil || n < 0 {
			return storage.ListPage[T]{}, fmt.Errorf("%w: invalid page token", storage.ErrInvalidInput)
		}
		offset = n
	}

	if offset >= len(items) {
		return storage.ListPage[T]{}, nil
	}

	end := offset + size
	var nextToken string
	if end < len(items) {
		nextToken = strconv.Itoa(end)
	} else {
		end = len(items)
	}

	clones := make([]T, end-offset)
	for i, v := range items[offset:end] {
		clones[i] = proto.Clone(v).(T)
	}
	return storage.ListPage[T]{Items: clones, NextPageToken: nextToken}, nil
}

// inDateRange reports whether ts (YYYY-MM-DD or RFC3339) falls within [start, end].
// Comparison is purely lexicographic on the date prefix (first 10 chars), which
// is valid for both YYYY-MM-DD and RFC3339 strings.
func inDateRange(ts, start, end string) bool {
	if len(ts) >= 10 {
		ts = ts[:10]
	}
	if start != "" && ts < start {
		return false
	}
	if end != "" && ts > end {
		return false
	}
	return true
}

// ---- UserProfile ------------------------------------------------------------

type userProfileStore struct {
	mu   sync.RWMutex
	data map[string]*v1.UserProfile
}

func (s *userProfileStore) GetByID(_ context.Context, id string) (*v1.UserProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return proto.Clone(p).(*v1.UserProfile), nil
}

func (s *userProfileStore) Upsert(_ context.Context, profile *v1.UserProfile) error {
	if profile.GetName() == "" {
		return fmt.Errorf("%w: profile name is required", storage.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[profile.GetName()] = proto.Clone(profile).(*v1.UserProfile)
	return nil
}

// ---- BleedingObservation ----------------------------------------------------

type bleedingStore struct {
	mu   sync.RWMutex
	data map[string]*v1.BleedingObservation
}

func (s *bleedingStore) Create(_ context.Context, obs *v1.BleedingObservation) error {
	if obs.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[obs.GetName()]; exists {
		return fmt.Errorf("%w: bleeding observation %s", storage.ErrConflict, obs.GetName())
	}
	s.data[obs.GetName()] = proto.Clone(obs).(*v1.BleedingObservation)
	return nil
}

func (s *bleedingStore) GetByID(_ context.Context, id string) (*v1.BleedingObservation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obs, ok := s.data[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return proto.Clone(obs).(*v1.BleedingObservation), nil
}

func (s *bleedingStore) ListByUserAndDateRange(_ context.Context, userID, start, end string, page storage.PageRequest) (storage.ListPage[*v1.BleedingObservation], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []*v1.BleedingObservation
	for _, obs := range s.data {
		if obs.GetUserId() != userID {
			continue
		}
		ts := obs.GetTimestamp().GetValue()
		if !inDateRange(ts, start, end) {
			continue
		}
		filtered = append(filtered, obs)
	}
	sortByTimestamp(filtered)
	return pageItems(filtered, page)
}

func (s *bleedingStore) DeleteByID(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return storage.ErrNotFound
	}
	delete(s.data, id)
	return nil
}

// ---- SymptomObservation -----------------------------------------------------

type symptomStore struct {
	mu   sync.RWMutex
	data map[string]*v1.SymptomObservation
}

func (s *symptomStore) Create(_ context.Context, obs *v1.SymptomObservation) error {
	if obs.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[obs.GetName()]; exists {
		return fmt.Errorf("%w: symptom observation %s", storage.ErrConflict, obs.GetName())
	}
	s.data[obs.GetName()] = proto.Clone(obs).(*v1.SymptomObservation)
	return nil
}

func (s *symptomStore) GetByID(_ context.Context, id string) (*v1.SymptomObservation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obs, ok := s.data[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return proto.Clone(obs).(*v1.SymptomObservation), nil
}

func (s *symptomStore) ListByUserAndDateRange(_ context.Context, userID, start, end string, page storage.PageRequest) (storage.ListPage[*v1.SymptomObservation], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []*v1.SymptomObservation
	for _, obs := range s.data {
		if obs.GetUserId() != userID {
			continue
		}
		if !inDateRange(obs.GetTimestamp().GetValue(), start, end) {
			continue
		}
		filtered = append(filtered, obs)
	}
	sortSymptomsByTimestamp(filtered)
	return pageItems(filtered, page)
}

func (s *symptomStore) DeleteByID(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return storage.ErrNotFound
	}
	delete(s.data, id)
	return nil
}

// ---- MoodObservation --------------------------------------------------------

type moodStore struct {
	mu   sync.RWMutex
	data map[string]*v1.MoodObservation
}

func (s *moodStore) Create(_ context.Context, obs *v1.MoodObservation) error {
	if obs.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[obs.GetName()]; exists {
		return fmt.Errorf("%w: mood observation %s", storage.ErrConflict, obs.GetName())
	}
	s.data[obs.GetName()] = proto.Clone(obs).(*v1.MoodObservation)
	return nil
}

func (s *moodStore) GetByID(_ context.Context, id string) (*v1.MoodObservation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obs, ok := s.data[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return proto.Clone(obs).(*v1.MoodObservation), nil
}

func (s *moodStore) ListByUserAndDateRange(_ context.Context, userID, start, end string, page storage.PageRequest) (storage.ListPage[*v1.MoodObservation], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []*v1.MoodObservation
	for _, obs := range s.data {
		if obs.GetUserId() != userID {
			continue
		}
		if !inDateRange(obs.GetTimestamp().GetValue(), start, end) {
			continue
		}
		filtered = append(filtered, obs)
	}
	sortMoodsByTimestamp(filtered)
	return pageItems(filtered, page)
}

func (s *moodStore) DeleteByID(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return storage.ErrNotFound
	}
	delete(s.data, id)
	return nil
}

// ---- Medication -------------------------------------------------------------

type medicationStore struct {
	mu   sync.RWMutex
	data map[string]*v1.Medication
}

func (s *medicationStore) Create(_ context.Context, med *v1.Medication) error {
	if med.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[med.GetName()]; exists {
		return fmt.Errorf("%w: medication %s", storage.ErrConflict, med.GetName())
	}
	s.data[med.GetName()] = proto.Clone(med).(*v1.Medication)
	return nil
}

func (s *medicationStore) GetByID(_ context.Context, id string) (*v1.Medication, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	med, ok := s.data[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return proto.Clone(med).(*v1.Medication), nil
}

func (s *medicationStore) ListByUser(_ context.Context, userID string, page storage.PageRequest) (storage.ListPage[*v1.Medication], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []*v1.Medication
	for _, med := range s.data {
		if med.GetUserId() == userID {
			filtered = append(filtered, med)
		}
	}
	sortMedicationsByID(filtered)
	return pageItems(filtered, page)
}

func (s *medicationStore) Update(_ context.Context, med *v1.Medication) error {
	if med.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[med.GetName()]; !ok {
		return storage.ErrNotFound
	}
	s.data[med.GetName()] = proto.Clone(med).(*v1.Medication)
	return nil
}

func (s *medicationStore) DeleteByID(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return storage.ErrNotFound
	}
	delete(s.data, id)
	return nil
}

// ---- MedicationEvent --------------------------------------------------------

type medicationEventStore struct {
	mu   sync.RWMutex
	data map[string]*v1.MedicationEvent
}

func (s *medicationEventStore) Create(_ context.Context, ev *v1.MedicationEvent) error {
	if ev.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[ev.GetName()]; exists {
		return fmt.Errorf("%w: medication event %s", storage.ErrConflict, ev.GetName())
	}
	s.data[ev.GetName()] = proto.Clone(ev).(*v1.MedicationEvent)
	return nil
}

func (s *medicationEventStore) GetByID(_ context.Context, id string) (*v1.MedicationEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ev, ok := s.data[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return proto.Clone(ev).(*v1.MedicationEvent), nil
}

func (s *medicationEventStore) ListByUserAndDateRange(_ context.Context, userID, start, end string, page storage.PageRequest) (storage.ListPage[*v1.MedicationEvent], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []*v1.MedicationEvent
	for _, ev := range s.data {
		if ev.GetUserId() != userID {
			continue
		}
		if !inDateRange(ev.GetTimestamp().GetValue(), start, end) {
			continue
		}
		filtered = append(filtered, ev)
	}
	sortMedicationEventsByTimestamp(filtered)
	return pageItems(filtered, page)
}

func (s *medicationEventStore) ListByMedicationID(_ context.Context, medicationID string, page storage.PageRequest) (storage.ListPage[*v1.MedicationEvent], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []*v1.MedicationEvent
	for _, ev := range s.data {
		if ev.GetMedicationId() == medicationID {
			filtered = append(filtered, ev)
		}
	}
	sortMedicationEventsByTimestamp(filtered)
	return pageItems(filtered, page)
}

func (s *medicationEventStore) DeleteByID(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return storage.ErrNotFound
	}
	delete(s.data, id)
	return nil
}

// ---- Cycle ------------------------------------------------------------------

type cycleStore struct {
	mu   sync.RWMutex
	data map[string]*v1.Cycle
}

func (s *cycleStore) Create(_ context.Context, cycle *v1.Cycle) error {
	if cycle.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[cycle.GetName()]; exists {
		return fmt.Errorf("%w: cycle %s", storage.ErrConflict, cycle.GetName())
	}
	s.data[cycle.GetName()] = proto.Clone(cycle).(*v1.Cycle)
	return nil
}

func (s *cycleStore) GetByID(_ context.Context, id string) (*v1.Cycle, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.data[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return proto.Clone(c).(*v1.Cycle), nil
}

func (s *cycleStore) ListByUser(_ context.Context, userID string, page storage.PageRequest) (storage.ListPage[*v1.Cycle], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []*v1.Cycle
	for _, c := range s.data {
		if c.GetUserId() == userID {
			filtered = append(filtered, c)
		}
	}
	sortCyclesByStartDate(filtered)
	return pageItems(filtered, page)
}

func (s *cycleStore) ListByUserAndDateRange(_ context.Context, userID, start, end string, page storage.PageRequest) (storage.ListPage[*v1.Cycle], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []*v1.Cycle
	for _, c := range s.data {
		if c.GetUserId() != userID {
			continue
		}
		cStart := c.GetStartDate().GetValue()
		cEnd := c.GetEndDate().GetValue()
		// Include cycle if its range overlaps with [start, end].
		if end != "" && cStart != "" && cStart > end {
			continue
		}
		if start != "" && cEnd != "" && cEnd < start {
			continue
		}
		filtered = append(filtered, c)
	}
	sortCyclesByStartDate(filtered)
	return pageItems(filtered, page)
}

func (s *cycleStore) Update(_ context.Context, cycle *v1.Cycle) error {
	if cycle.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[cycle.GetName()]; !ok {
		return storage.ErrNotFound
	}
	s.data[cycle.GetName()] = proto.Clone(cycle).(*v1.Cycle)
	return nil
}

func (s *cycleStore) DeleteByID(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return storage.ErrNotFound
	}
	delete(s.data, id)
	return nil
}

// ---- PhaseEstimate ----------------------------------------------------------

type phaseEstimateStore struct {
	mu   sync.RWMutex
	data map[string]*v1.PhaseEstimate
}

func (s *phaseEstimateStore) Create(_ context.Context, est *v1.PhaseEstimate) error {
	if est.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[est.GetName()]; exists {
		return fmt.Errorf("%w: phase estimate %s", storage.ErrConflict, est.GetName())
	}
	s.data[est.GetName()] = proto.Clone(est).(*v1.PhaseEstimate)
	return nil
}

func (s *phaseEstimateStore) ListByUserAndDateRange(_ context.Context, userID, start, end string, page storage.PageRequest) (storage.ListPage[*v1.PhaseEstimate], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []*v1.PhaseEstimate
	for _, est := range s.data {
		if est.GetUserId() != userID {
			continue
		}
		if !inDateRange(est.GetDate().GetValue(), start, end) {
			continue
		}
		filtered = append(filtered, est)
	}
	sortPhaseEstimatesByDate(filtered)
	return pageItems(filtered, page)
}

func (s *phaseEstimateStore) DeleteByCycleID(_ context.Context, cycleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, est := range s.data {
		for _, ref := range est.GetBasedOnRecordRefs() {
			if ref.GetName() == cycleID {
				delete(s.data, id)
				break
			}
		}
	}
	return nil
}

// ---- Prediction -------------------------------------------------------------

type predictionStore struct {
	mu   sync.RWMutex
	data map[string]*v1.Prediction
}

func (s *predictionStore) Create(_ context.Context, pred *v1.Prediction) error {
	if pred.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[pred.GetName()]; exists {
		return fmt.Errorf("%w: prediction %s", storage.ErrConflict, pred.GetName())
	}
	s.data[pred.GetName()] = proto.Clone(pred).(*v1.Prediction)
	return nil
}

func (s *predictionStore) ListByUser(_ context.Context, userID string, page storage.PageRequest) (storage.ListPage[*v1.Prediction], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []*v1.Prediction
	for _, pred := range s.data {
		if pred.GetUserId() == userID {
			filtered = append(filtered, pred)
		}
	}
	sortPredictionsByStartDate(filtered)
	return pageItems(filtered, page)
}

func (s *predictionStore) DeleteByUser(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, pred := range s.data {
		if pred.GetUserId() == userID {
			delete(s.data, id)
		}
	}
	return nil
}

// ---- Insight ----------------------------------------------------------------

type insightStore struct {
	mu   sync.RWMutex
	data map[string]*v1.Insight
}

func (s *insightStore) Create(_ context.Context, insight *v1.Insight) error {
	if insight.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[insight.GetName()]; exists {
		return fmt.Errorf("%w: insight %s", storage.ErrConflict, insight.GetName())
	}
	s.data[insight.GetName()] = proto.Clone(insight).(*v1.Insight)
	return nil
}

func (s *insightStore) ListByUser(_ context.Context, userID string, page storage.PageRequest) (storage.ListPage[*v1.Insight], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []*v1.Insight
	for _, insight := range s.data {
		if insight.GetUserId() == userID {
			filtered = append(filtered, insight)
		}
	}
	sortInsightsByID(filtered)
	return pageItems(filtered, page)
}

func (s *insightStore) DeleteByUser(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, insight := range s.data {
		if insight.GetUserId() == userID {
			delete(s.data, id)
		}
	}
	return nil
}
