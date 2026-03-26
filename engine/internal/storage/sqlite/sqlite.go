// Package sqlite provides the SQLite-backed implementation of
// the storage.Repository interface. All proto messages are stored
// as serialized protobuf blobs alongside indexed scalar columns.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	"github.com/2ajoyce/openmenses/engine/internal/storage/migrations"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

const defaultPageSize = 50

// Store is the SQLite-backed repository.
type Store struct {
	db               *sql.DB
	profiles         *userProfileRepo
	bleedings        *bleedingRepo
	symptoms         *symptomRepo
	moods            *moodRepo
	medications      *medicationRepo
	medicationEvents *medicationEventRepo
	cycles           *cycleRepo
	phaseEstimates   *phaseEstimateRepo
	predictions      *predictionRepo
	insights         *insightRepo
}

// Open opens (or creates) the SQLite database at path and runs all pending
// migrations. Use ":memory:" for an in-memory test database.
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	// Single writer to avoid SQLITE_BUSY on concurrent access.
	db.SetMaxOpenConns(1)

	if err := migrations.Run(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite: migrate: %w", err)
	}

	s := &Store{db: db}
	s.profiles = &userProfileRepo{db: db}
	s.bleedings = &bleedingRepo{db: db}
	s.symptoms = &symptomRepo{db: db}
	s.moods = &moodRepo{db: db}
	s.medications = &medicationRepo{db: db}
	s.medicationEvents = &medicationEventRepo{db: db}
	s.cycles = &cycleRepo{db: db}
	s.phaseEstimates = &phaseEstimateRepo{db: db}
	s.predictions = &predictionRepo{db: db}
	s.insights = &insightRepo{db: db}
	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error { return s.db.Close() }

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

func pageArgs(page storage.PageRequest) (limit, offset int, err error) {
	limit = int(page.PageSize)
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > 500 {
		limit = 500
	}
	if page.PageToken != "" {
		offset, err = strconv.Atoi(page.PageToken)
		if err != nil || offset < 0 {
			return 0, 0, fmt.Errorf("%w: invalid page token", storage.ErrInvalidInput)
		}
	}
	return limit, offset, nil
}

func nextToken(offset, limit, total int) string {
	next := offset + limit
	if next >= total {
		return ""
	}
	return strconv.Itoa(next)
}

func marshal(m proto.Message) ([]byte, error) {
	return proto.Marshal(m)
}

func mapNotFound(err error) error {
	if err == sql.ErrNoRows {
		return storage.ErrNotFound
	}
	return err
}

// ---- UserProfile ------------------------------------------------------------

type userProfileRepo struct{ db *sql.DB }

func (r *userProfileRepo) GetByID(ctx context.Context, id string) (*v1.UserProfile, error) {
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT data FROM user_profiles WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return nil, mapNotFound(err)
	}
	var p v1.UserProfile
	if err := proto.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *userProfileRepo) Create(ctx context.Context, profile *v1.UserProfile) error {
	if profile.GetName() == "" {
		return fmt.Errorf("%w: profile name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(profile)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO user_profiles (id, data) VALUES (?, ?)`,
		profile.GetName(), data)
	if isConflict(err) {
		return fmt.Errorf("%w: user profile %s", storage.ErrConflict, profile.GetName())
	}
	return err
}

func (r *userProfileRepo) Update(ctx context.Context, profile *v1.UserProfile) error {
	if profile.GetName() == "" {
		return fmt.Errorf("%w: profile name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(profile)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx,
		`UPDATE user_profiles SET data = ? WHERE id = ?`,
		data, profile.GetName())
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *userProfileRepo) Upsert(ctx context.Context, profile *v1.UserProfile) error {
	if profile.GetName() == "" {
		return fmt.Errorf("%w: profile name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(profile)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO user_profiles (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		profile.GetName(), data)
	return err
}

func (r *userProfileRepo) DeleteByID(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM user_profiles WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: profile %s", storage.ErrNotFound, id)
	}
	return nil
}

// ---- BleedingObservation ----------------------------------------------------

type bleedingRepo struct{ db *sql.DB }

func (r *bleedingRepo) Create(ctx context.Context, obs *v1.BleedingObservation) error {
	if obs.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(obs)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO bleeding_observations (id, user_id, timestamp, data) VALUES (?, ?, ?, ?)`,
		obs.GetName(), obs.GetUserId(), obs.GetTimestamp().GetValue(), data)
	if isConflict(err) {
		return fmt.Errorf("%w: bleeding observation %s", storage.ErrConflict, obs.GetName())
	}
	return err
}

func (r *bleedingRepo) GetByID(ctx context.Context, id string) (*v1.BleedingObservation, error) {
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT data FROM bleeding_observations WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return nil, mapNotFound(err)
	}
	var obs v1.BleedingObservation
	return &obs, proto.Unmarshal(data, &obs)
}

func (r *bleedingRepo) ListByUserAndDateRange(ctx context.Context, userID, start, end string, page storage.PageRequest) (storage.ListPage[*v1.BleedingObservation], error) {
	limit, offset, err := pageArgs(page)
	if err != nil {
		return storage.ListPage[*v1.BleedingObservation]{}, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM bleeding_observations
		 WHERE user_id = ?
		   AND (? = '' OR substr(timestamp,1,10) >= ?)
		   AND (? = '' OR substr(timestamp,1,10) <= ?)
		 ORDER BY timestamp
		 LIMIT ? OFFSET ?`,
		userID, start, start, end, end, limit+1, offset)
	if err != nil {
		return storage.ListPage[*v1.BleedingObservation]{}, err
	}
	defer func() { _ = rows.Close() }()
	return scanBleedings(rows, limit, offset)
}

func (r *bleedingRepo) DeleteByID(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM bleeding_observations WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (r *bleedingRepo) DeleteByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM bleeding_observations WHERE user_id = ?`, userID)
	return err
}

func scanBleedings(rows *sql.Rows, limit, offset int) (storage.ListPage[*v1.BleedingObservation], error) {
	var items []*v1.BleedingObservation
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return storage.ListPage[*v1.BleedingObservation]{}, err
		}
		var obs v1.BleedingObservation
		if err := proto.Unmarshal(data, &obs); err != nil {
			return storage.ListPage[*v1.BleedingObservation]{}, err
		}
		items = append(items, &obs)
	}
	if err := rows.Err(); err != nil {
		return storage.ListPage[*v1.BleedingObservation]{}, err
	}
	var token string
	if len(items) > limit {
		items = items[:limit]
		token = nextToken(offset, limit, offset+limit+1)
	}
	return storage.ListPage[*v1.BleedingObservation]{Items: items, NextPageToken: token}, nil
}

// ---- SymptomObservation -----------------------------------------------------

type symptomRepo struct{ db *sql.DB }

func (r *symptomRepo) Create(ctx context.Context, obs *v1.SymptomObservation) error {
	if obs.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(obs)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO symptom_observations (id, user_id, timestamp, data) VALUES (?, ?, ?, ?)`,
		obs.GetName(), obs.GetUserId(), obs.GetTimestamp().GetValue(), data)
	if isConflict(err) {
		return fmt.Errorf("%w: symptom observation %s", storage.ErrConflict, obs.GetName())
	}
	return err
}

func (r *symptomRepo) GetByID(ctx context.Context, id string) (*v1.SymptomObservation, error) {
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT data FROM symptom_observations WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return nil, mapNotFound(err)
	}
	var obs v1.SymptomObservation
	return &obs, proto.Unmarshal(data, &obs)
}

func (r *symptomRepo) ListByUser(ctx context.Context, userID string, page storage.PageRequest) (storage.ListPage[*v1.SymptomObservation], error) {
	limit, offset, err := pageArgs(page)
	if err != nil {
		return storage.ListPage[*v1.SymptomObservation]{}, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM symptom_observations WHERE user_id = ? ORDER BY timestamp LIMIT ? OFFSET ?`,
		userID, limit+1, offset)
	if err != nil {
		return storage.ListPage[*v1.SymptomObservation]{}, err
	}
	defer func() { _ = rows.Close() }()
	return scanSymptoms(rows, limit, offset)
}

func (r *symptomRepo) ListByUserAndDateRange(ctx context.Context, userID, start, end string, page storage.PageRequest) (storage.ListPage[*v1.SymptomObservation], error) {
	limit, offset, err := pageArgs(page)
	if err != nil {
		return storage.ListPage[*v1.SymptomObservation]{}, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM symptom_observations
		 WHERE user_id = ?
		   AND (? = '' OR substr(timestamp,1,10) >= ?)
		   AND (? = '' OR substr(timestamp,1,10) <= ?)
		 ORDER BY timestamp
		 LIMIT ? OFFSET ?`,
		userID, start, start, end, end, limit+1, offset)
	if err != nil {
		return storage.ListPage[*v1.SymptomObservation]{}, err
	}
	defer func() { _ = rows.Close() }()
	return scanSymptoms(rows, limit, offset)
}

func (r *symptomRepo) DeleteByID(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM symptom_observations WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (r *symptomRepo) DeleteByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM symptom_observations WHERE user_id = ?`, userID)
	return err
}

func scanSymptoms(rows *sql.Rows, limit, offset int) (storage.ListPage[*v1.SymptomObservation], error) {
	var items []*v1.SymptomObservation
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return storage.ListPage[*v1.SymptomObservation]{}, err
		}
		var obs v1.SymptomObservation
		if err := proto.Unmarshal(data, &obs); err != nil {
			return storage.ListPage[*v1.SymptomObservation]{}, err
		}
		items = append(items, &obs)
	}
	if err := rows.Err(); err != nil {
		return storage.ListPage[*v1.SymptomObservation]{}, err
	}
	var token string
	if len(items) > limit {
		items = items[:limit]
		token = nextToken(offset, limit, offset+limit+1)
	}
	return storage.ListPage[*v1.SymptomObservation]{Items: items, NextPageToken: token}, nil
}

// ---- MoodObservation --------------------------------------------------------

type moodRepo struct{ db *sql.DB }

func (r *moodRepo) Create(ctx context.Context, obs *v1.MoodObservation) error {
	if obs.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(obs)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO mood_observations (id, user_id, timestamp, data) VALUES (?, ?, ?, ?)`,
		obs.GetName(), obs.GetUserId(), obs.GetTimestamp().GetValue(), data)
	if isConflict(err) {
		return fmt.Errorf("%w: mood observation %s", storage.ErrConflict, obs.GetName())
	}
	return err
}

func (r *moodRepo) GetByID(ctx context.Context, id string) (*v1.MoodObservation, error) {
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT data FROM mood_observations WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return nil, mapNotFound(err)
	}
	var obs v1.MoodObservation
	return &obs, proto.Unmarshal(data, &obs)
}

func (r *moodRepo) ListByUserAndDateRange(ctx context.Context, userID, start, end string, page storage.PageRequest) (storage.ListPage[*v1.MoodObservation], error) {
	limit, offset, err := pageArgs(page)
	if err != nil {
		return storage.ListPage[*v1.MoodObservation]{}, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM mood_observations
		 WHERE user_id = ?
		   AND (? = '' OR substr(timestamp,1,10) >= ?)
		   AND (? = '' OR substr(timestamp,1,10) <= ?)
		 ORDER BY timestamp
		 LIMIT ? OFFSET ?`,
		userID, start, start, end, end, limit+1, offset)
	if err != nil {
		return storage.ListPage[*v1.MoodObservation]{}, err
	}
	defer func() { _ = rows.Close() }()
	return scanMoods(rows, limit, offset)
}

func (r *moodRepo) DeleteByID(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM mood_observations WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (r *moodRepo) DeleteByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mood_observations WHERE user_id = ?`, userID)
	return err
}

func scanMoods(rows *sql.Rows, limit, offset int) (storage.ListPage[*v1.MoodObservation], error) {
	var items []*v1.MoodObservation
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return storage.ListPage[*v1.MoodObservation]{}, err
		}
		var obs v1.MoodObservation
		if err := proto.Unmarshal(data, &obs); err != nil {
			return storage.ListPage[*v1.MoodObservation]{}, err
		}
		items = append(items, &obs)
	}
	if err := rows.Err(); err != nil {
		return storage.ListPage[*v1.MoodObservation]{}, err
	}
	var token string
	if len(items) > limit {
		items = items[:limit]
		token = nextToken(offset, limit, offset+limit+1)
	}
	return storage.ListPage[*v1.MoodObservation]{Items: items, NextPageToken: token}, nil
}

// ---- Medication -------------------------------------------------------------

type medicationRepo struct{ db *sql.DB }

func (r *medicationRepo) Create(ctx context.Context, med *v1.Medication) error {
	if med.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(med)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO medications (id, user_id, data) VALUES (?, ?, ?)`,
		med.GetName(), med.GetUserId(), data)
	if isConflict(err) {
		return fmt.Errorf("%w: medication %s", storage.ErrConflict, med.GetName())
	}
	return err
}

func (r *medicationRepo) GetByID(ctx context.Context, id string) (*v1.Medication, error) {
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT data FROM medications WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return nil, mapNotFound(err)
	}
	var med v1.Medication
	return &med, proto.Unmarshal(data, &med)
}

func (r *medicationRepo) ListByUser(ctx context.Context, userID string, page storage.PageRequest) (storage.ListPage[*v1.Medication], error) {
	limit, offset, err := pageArgs(page)
	if err != nil {
		return storage.ListPage[*v1.Medication]{}, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM medications WHERE user_id = ? ORDER BY id LIMIT ? OFFSET ?`,
		userID, limit+1, offset)
	if err != nil {
		return storage.ListPage[*v1.Medication]{}, err
	}
	defer func() { _ = rows.Close() }()
	var items []*v1.Medication
	for rows.Next() {
		var d []byte
		if err := rows.Scan(&d); err != nil {
			return storage.ListPage[*v1.Medication]{}, err
		}
		var m v1.Medication
		if err := proto.Unmarshal(d, &m); err != nil {
			return storage.ListPage[*v1.Medication]{}, err
		}
		items = append(items, &m)
	}
	if err := rows.Err(); err != nil {
		return storage.ListPage[*v1.Medication]{}, err
	}
	var token string
	if len(items) > limit {
		items = items[:limit]
		token = nextToken(offset, limit, offset+limit+1)
	}
	return storage.ListPage[*v1.Medication]{Items: items, NextPageToken: token}, nil
}

func (r *medicationRepo) Update(ctx context.Context, med *v1.Medication) error {
	if med.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(med)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE medications SET data = ?, user_id = ? WHERE id = ?`,
		data, med.GetUserId(), med.GetName())
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (r *medicationRepo) DeleteByID(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM medications WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (r *medicationRepo) DeleteByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM medications WHERE user_id = ?`, userID)
	return err
}

// ---- MedicationEvent --------------------------------------------------------

type medicationEventRepo struct{ db *sql.DB }

func (r *medicationEventRepo) Create(ctx context.Context, ev *v1.MedicationEvent) error {
	if ev.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(ev)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO medication_events (id, user_id, medication_id, timestamp, data) VALUES (?, ?, ?, ?, ?)`,
		ev.GetName(), ev.GetUserId(), ev.GetMedicationId(), ev.GetTimestamp().GetValue(), data)
	if isConflict(err) {
		return fmt.Errorf("%w: medication event %s", storage.ErrConflict, ev.GetName())
	}
	return err
}

func (r *medicationEventRepo) GetByID(ctx context.Context, id string) (*v1.MedicationEvent, error) {
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT data FROM medication_events WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return nil, mapNotFound(err)
	}
	var ev v1.MedicationEvent
	return &ev, proto.Unmarshal(data, &ev)
}

func (r *medicationEventRepo) ListByUserAndDateRange(ctx context.Context, userID, start, end string, page storage.PageRequest) (storage.ListPage[*v1.MedicationEvent], error) {
	limit, offset, err := pageArgs(page)
	if err != nil {
		return storage.ListPage[*v1.MedicationEvent]{}, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM medication_events
		 WHERE user_id = ?
		   AND (? = '' OR substr(timestamp,1,10) >= ?)
		   AND (? = '' OR substr(timestamp,1,10) <= ?)
		 ORDER BY timestamp
		 LIMIT ? OFFSET ?`,
		userID, start, start, end, end, limit+1, offset)
	if err != nil {
		return storage.ListPage[*v1.MedicationEvent]{}, err
	}
	defer func() { _ = rows.Close() }()
	return scanMedEvents(rows, limit, offset)
}

func (r *medicationEventRepo) ListByMedicationID(ctx context.Context, medicationID string, page storage.PageRequest) (storage.ListPage[*v1.MedicationEvent], error) {
	limit, offset, err := pageArgs(page)
	if err != nil {
		return storage.ListPage[*v1.MedicationEvent]{}, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM medication_events WHERE medication_id = ? ORDER BY timestamp LIMIT ? OFFSET ?`,
		medicationID, limit+1, offset)
	if err != nil {
		return storage.ListPage[*v1.MedicationEvent]{}, err
	}
	defer func() { _ = rows.Close() }()
	return scanMedEvents(rows, limit, offset)
}

func (r *medicationEventRepo) DeleteByID(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM medication_events WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (r *medicationEventRepo) DeleteByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM medication_events WHERE user_id = ?`, userID)
	return err
}

func scanMedEvents(rows *sql.Rows, limit, offset int) (storage.ListPage[*v1.MedicationEvent], error) {
	var items []*v1.MedicationEvent
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return storage.ListPage[*v1.MedicationEvent]{}, err
		}
		var ev v1.MedicationEvent
		if err := proto.Unmarshal(data, &ev); err != nil {
			return storage.ListPage[*v1.MedicationEvent]{}, err
		}
		items = append(items, &ev)
	}
	if err := rows.Err(); err != nil {
		return storage.ListPage[*v1.MedicationEvent]{}, err
	}
	var token string
	if len(items) > limit {
		items = items[:limit]
		token = nextToken(offset, limit, offset+limit+1)
	}
	return storage.ListPage[*v1.MedicationEvent]{Items: items, NextPageToken: token}, nil
}

// ---- Cycle ------------------------------------------------------------------

type cycleRepo struct{ db *sql.DB }

func (r *cycleRepo) Create(ctx context.Context, cycle *v1.Cycle) error {
	if cycle.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(cycle)
	if err != nil {
		return err
	}
	endDate := cycle.GetEndDate().GetValue()
	var endArg interface{} = endDate
	if endDate == "" {
		endArg = nil
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO cycles (id, user_id, start_date, end_date, data) VALUES (?, ?, ?, ?, ?)`,
		cycle.GetName(), cycle.GetUserId(), cycle.GetStartDate().GetValue(), endArg, data)
	if isConflict(err) {
		return fmt.Errorf("%w: cycle %s", storage.ErrConflict, cycle.GetName())
	}
	return err
}

func (r *cycleRepo) GetByID(ctx context.Context, id string) (*v1.Cycle, error) {
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT data FROM cycles WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return nil, mapNotFound(err)
	}
	var c v1.Cycle
	return &c, proto.Unmarshal(data, &c)
}

func (r *cycleRepo) ListByUser(ctx context.Context, userID string, page storage.PageRequest) (storage.ListPage[*v1.Cycle], error) {
	limit, offset, err := pageArgs(page)
	if err != nil {
		return storage.ListPage[*v1.Cycle]{}, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM cycles WHERE user_id = ? ORDER BY start_date LIMIT ? OFFSET ?`,
		userID, limit+1, offset)
	if err != nil {
		return storage.ListPage[*v1.Cycle]{}, err
	}
	defer func() { _ = rows.Close() }()
	return scanCycles(rows, limit, offset)
}

func (r *cycleRepo) ListByUserAndDateRange(ctx context.Context, userID, start, end string, page storage.PageRequest) (storage.ListPage[*v1.Cycle], error) {
	limit, offset, err := pageArgs(page)
	if err != nil {
		return storage.ListPage[*v1.Cycle]{}, err
	}
	// Include cycles whose range overlaps with [start, end].
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM cycles
		 WHERE user_id = ?
		   AND (? = '' OR start_date <= ?)
		   AND (? = '' OR end_date IS NULL OR end_date >= ?)
		 ORDER BY start_date
		 LIMIT ? OFFSET ?`,
		userID, end, end, start, start, limit+1, offset)
	if err != nil {
		return storage.ListPage[*v1.Cycle]{}, err
	}
	defer func() { _ = rows.Close() }()
	return scanCycles(rows, limit, offset)
}

func (r *cycleRepo) Update(ctx context.Context, cycle *v1.Cycle) error {
	if cycle.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(cycle)
	if err != nil {
		return err
	}
	endDate := cycle.GetEndDate().GetValue()
	var endArg interface{} = endDate
	if endDate == "" {
		endArg = nil
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE cycles SET start_date = ?, end_date = ?, data = ?, user_id = ? WHERE id = ?`,
		cycle.GetStartDate().GetValue(), endArg, data, cycle.GetUserId(), cycle.GetName())
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (r *cycleRepo) DeleteByID(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM cycles WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (r *cycleRepo) DeleteByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cycles WHERE user_id = ?`, userID)
	return err
}

func scanCycles(rows *sql.Rows, limit, offset int) (storage.ListPage[*v1.Cycle], error) {
	var items []*v1.Cycle
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return storage.ListPage[*v1.Cycle]{}, err
		}
		var c v1.Cycle
		if err := proto.Unmarshal(data, &c); err != nil {
			return storage.ListPage[*v1.Cycle]{}, err
		}
		items = append(items, &c)
	}
	if err := rows.Err(); err != nil {
		return storage.ListPage[*v1.Cycle]{}, err
	}
	var token string
	if len(items) > limit {
		items = items[:limit]
		token = nextToken(offset, limit, offset+limit+1)
	}
	return storage.ListPage[*v1.Cycle]{Items: items, NextPageToken: token}, nil
}

// ---- PhaseEstimate ----------------------------------------------------------

type phaseEstimateRepo struct{ db *sql.DB }

func (r *phaseEstimateRepo) Create(ctx context.Context, est *v1.PhaseEstimate) error {
	if est.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(est)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO phase_estimates (id, user_id, date, data) VALUES (?, ?, ?, ?)`,
		est.GetName(), est.GetUserId(), est.GetDate().GetValue(), data)
	if isConflict(err) {
		return fmt.Errorf("%w: phase estimate %s", storage.ErrConflict, est.GetName())
	}
	return err
}

func (r *phaseEstimateRepo) ListByUserAndDateRange(ctx context.Context, userID, start, end string, page storage.PageRequest) (storage.ListPage[*v1.PhaseEstimate], error) {
	limit, offset, err := pageArgs(page)
	if err != nil {
		return storage.ListPage[*v1.PhaseEstimate]{}, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM phase_estimates
		 WHERE user_id = ?
		   AND (? = '' OR date >= ?)
		   AND (? = '' OR date <= ?)
		 ORDER BY date
		 LIMIT ? OFFSET ?`,
		userID, start, start, end, end, limit+1, offset)
	if err != nil {
		return storage.ListPage[*v1.PhaseEstimate]{}, err
	}
	defer func() { _ = rows.Close() }()
	var items []*v1.PhaseEstimate
	for rows.Next() {
		var d []byte
		if err := rows.Scan(&d); err != nil {
			return storage.ListPage[*v1.PhaseEstimate]{}, err
		}
		var est v1.PhaseEstimate
		if err := proto.Unmarshal(d, &est); err != nil {
			return storage.ListPage[*v1.PhaseEstimate]{}, err
		}
		items = append(items, &est)
	}
	if err := rows.Err(); err != nil {
		return storage.ListPage[*v1.PhaseEstimate]{}, err
	}
	var token string
	if len(items) > limit {
		items = items[:limit]
		token = nextToken(offset, limit, offset+limit+1)
	}
	return storage.ListPage[*v1.PhaseEstimate]{Items: items, NextPageToken: token}, nil
}

func (r *phaseEstimateRepo) DeleteByCycleID(ctx context.Context, cycleID string) error {
	// Phase estimates store their associated cycle name in based_on_record_refs.
	// Because the refs are encoded in the blob, we load all estimates for the user
	// and delete those that reference the cycleID. This is acceptable because
	// DeleteByCycleID is a rare, write-heavy operation triggered by re-detection.
	rows, err := r.db.QueryContext(ctx, `SELECT id, data FROM phase_estimates`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	var toDelete []string
	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return err
		}
		var est v1.PhaseEstimate
		if err := proto.Unmarshal(data, &est); err != nil {
			return err
		}
		for _, ref := range est.GetBasedOnRecordRefs() {
			if ref.GetName() == cycleID {
				toDelete = append(toDelete, id)
				break
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, id := range toDelete {
		if _, err := r.db.ExecContext(ctx, `DELETE FROM phase_estimates WHERE id = ?`, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *phaseEstimateRepo) DeleteByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM phase_estimates WHERE user_id = ?`, userID)
	return err
}

// ---- Prediction -------------------------------------------------------------

type predictionRepo struct{ db *sql.DB }

func (r *predictionRepo) Create(ctx context.Context, pred *v1.Prediction) error {
	if pred.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(pred)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO predictions (id, user_id, data) VALUES (?, ?, ?)`,
		pred.GetName(), pred.GetUserId(), data)
	if isConflict(err) {
		return fmt.Errorf("%w: prediction %s", storage.ErrConflict, pred.GetName())
	}
	return err
}

func (r *predictionRepo) ListByUser(ctx context.Context, userID string, page storage.PageRequest) (storage.ListPage[*v1.Prediction], error) {
	limit, offset, err := pageArgs(page)
	if err != nil {
		return storage.ListPage[*v1.Prediction]{}, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM predictions WHERE user_id = ? ORDER BY id LIMIT ? OFFSET ?`,
		userID, limit+1, offset)
	if err != nil {
		return storage.ListPage[*v1.Prediction]{}, err
	}
	defer func() { _ = rows.Close() }()
	var items []*v1.Prediction
	for rows.Next() {
		var d []byte
		if err := rows.Scan(&d); err != nil {
			return storage.ListPage[*v1.Prediction]{}, err
		}
		var pred v1.Prediction
		if err := proto.Unmarshal(d, &pred); err != nil {
			return storage.ListPage[*v1.Prediction]{}, err
		}
		items = append(items, &pred)
	}
	if err := rows.Err(); err != nil {
		return storage.ListPage[*v1.Prediction]{}, err
	}
	var token string
	if len(items) > limit {
		items = items[:limit]
		token = nextToken(offset, limit, offset+limit+1)
	}
	return storage.ListPage[*v1.Prediction]{Items: items, NextPageToken: token}, nil
}

func (r *predictionRepo) DeleteByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM predictions WHERE user_id = ?`, userID)
	return err
}

// ---- Insight ----------------------------------------------------------------

type insightRepo struct{ db *sql.DB }

func (r *insightRepo) Create(ctx context.Context, insight *v1.Insight) error {
	if insight.GetName() == "" {
		return fmt.Errorf("%w: name is required", storage.ErrInvalidInput)
	}
	data, err := marshal(insight)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO insights (id, user_id, data) VALUES (?, ?, ?)`,
		insight.GetName(), insight.GetUserId(), data)
	if isConflict(err) {
		return fmt.Errorf("%w: insight %s", storage.ErrConflict, insight.GetName())
	}
	return err
}

func (r *insightRepo) ListByUser(ctx context.Context, userID string, page storage.PageRequest) (storage.ListPage[*v1.Insight], error) {
	limit, offset, err := pageArgs(page)
	if err != nil {
		return storage.ListPage[*v1.Insight]{}, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM insights WHERE user_id = ? ORDER BY id LIMIT ? OFFSET ?`,
		userID, limit+1, offset)
	if err != nil {
		return storage.ListPage[*v1.Insight]{}, err
	}
	defer func() { _ = rows.Close() }()
	var items []*v1.Insight
	for rows.Next() {
		var d []byte
		if err := rows.Scan(&d); err != nil {
			return storage.ListPage[*v1.Insight]{}, err
		}
		var ins v1.Insight
		if err := proto.Unmarshal(d, &ins); err != nil {
			return storage.ListPage[*v1.Insight]{}, err
		}
		items = append(items, &ins)
	}
	if err := rows.Err(); err != nil {
		return storage.ListPage[*v1.Insight]{}, err
	}
	var token string
	if len(items) > limit {
		items = items[:limit]
		token = nextToken(offset, limit, offset+limit+1)
	}
	return storage.ListPage[*v1.Insight]{Items: items, NextPageToken: token}, nil
}

func (r *insightRepo) DeleteByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM insights WHERE user_id = ?`, userID)
	return err
}

// ---- error helpers ----------------------------------------------------------

func isConflict(err error) bool {
	if err == nil {
		return false
	}
	// modernc/sqlite returns errors with "UNIQUE constraint failed" in the message.
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func requireAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}
