package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"github.com/2ajoyce/openmenses/engine/internal/service"
	"github.com/2ajoyce/openmenses/engine/internal/storage"
	"github.com/2ajoyce/openmenses/engine/internal/storage/memory"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

var ctx = context.Background()

// newSvc creates a CycleTrackerService backed by an empty in-memory store.
func newSvc(t *testing.T) *service.CycleTrackerService {
	t.Helper()
	svc, err := service.New(memory.New())
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

// newSvcWithStore creates a CycleTrackerService backed by the given store.
func newSvcWithStore(t *testing.T, store storage.Repository) *service.CycleTrackerService {
	t.Helper()
	svc, err := service.New(store)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

// codeOf extracts the Connect-RPC error code from an error, or returns
// connect.CodeUnknown if the error is not a Connect error.
func codeOf(err error) connect.Code {
	var connErr *connect.Error
	if errors.As(err, &connErr) {
		return connErr.Code()
	}
	return connect.CodeUnknown
}

// ─── helper builders ──────────────────────────────────────────────────────────

func validProfile(id string) *v1.UserProfile {
	return &v1.UserProfile{
		Id:               id,
		BiologicalCycle:  v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		Contraception:    v1.ContraceptionType_CONTRACEPTION_TYPE_NONE,
		CycleRegularity:  v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
		ReproductiveGoal: v1.ReproductiveGoal_REPRODUCTIVE_GOAL_PREGNANCY_IRRELEVANT,
		TrackingFocus:    []v1.TrackingFocus{v1.TrackingFocus_TRACKING_FOCUS_BLEEDING},
	}
}

func validBleeding(id, userID, date string) *v1.BleedingObservation {
	return &v1.BleedingObservation{
		Id:        id,
		UserId:    userID,
		Timestamp: &v1.DateTime{Value: date + "T10:00:00Z"},
		Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
	}
}

func validSymptom(id, userID, date string) *v1.SymptomObservation {
	return &v1.SymptomObservation{
		Id:        id,
		UserId:    userID,
		Timestamp: &v1.DateTime{Value: date + "T10:00:00Z"},
		Symptom:   v1.SymptomType_SYMPTOM_TYPE_CRAMPS,
	}
}

func validMood(id, userID, date string) *v1.MoodObservation {
	return &v1.MoodObservation{
		Id:        id,
		UserId:    userID,
		Timestamp: &v1.DateTime{Value: date + "T10:00:00Z"},
		Mood:      v1.MoodType_MOOD_TYPE_HAPPY,
	}
}

func validMedication(id, userID string) *v1.Medication {
	return &v1.Medication{
		Id:       id,
		UserId:   userID,
		Name:     "Ibuprofen",
		Category: v1.MedicationCategory_MEDICATION_CATEGORY_PAIN_RELIEF,
		Active:   true,
	}
}

func validMedEvent(id, userID, medID, date string) *v1.MedicationEvent {
	return &v1.MedicationEvent{
		Id:           id,
		UserId:       userID,
		MedicationId: medID,
		Timestamp:    &v1.DateTime{Value: date + "T10:00:00Z"},
		Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
	}
}

// ─── GetUserProfile ───────────────────────────────────────────────────────────

func TestGetUserProfile_NotFound(t *testing.T) {
	svc := newSvc(t)
	_, err := svc.GetUserProfile(ctx, connect.NewRequest(&v1.GetUserProfileRequest{UserId: "missing"}))
	if codeOf(err) != connect.CodeNotFound {
		t.Fatalf("want CodeNotFound, got %v", err)
	}
}

func TestGetUserProfile_Found(t *testing.T) {
	store := memory.New()
	if err := store.UserProfiles().Upsert(ctx, validProfile("u1")); err != nil {
		t.Fatal(err)
	}
	svc := newSvcWithStore(t, store)
	resp, err := svc.GetUserProfile(ctx, connect.NewRequest(&v1.GetUserProfileRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetProfile().GetId() != "u1" {
		t.Errorf("got profile id %q, want u1", resp.Msg.GetProfile().GetId())
	}
}

// ─── UpsertUserProfile ────────────────────────────────────────────────────────

func TestUpsertUserProfile_HappyPath(t *testing.T) {
	svc := newSvc(t)
	profile := validProfile("u1")
	resp, err := svc.UpsertUserProfile(ctx, connect.NewRequest(&v1.UpsertUserProfileRequest{Profile: profile}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetProfile().GetId() != "u1" {
		t.Errorf("got %q, want u1", resp.Msg.GetProfile().GetId())
	}
}

func TestUpsertUserProfile_ValidationFailure(t *testing.T) {
	svc := newSvc(t)
	// Missing required fields (no tracking_focus).
	bad := &v1.UserProfile{
		Id:               "u1",
		BiologicalCycle:  v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		Contraception:    v1.ContraceptionType_CONTRACEPTION_TYPE_NONE,
		CycleRegularity:  v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
		ReproductiveGoal: v1.ReproductiveGoal_REPRODUCTIVE_GOAL_PREGNANCY_IRRELEVANT,
		// TrackingFocus intentionally left empty → schema violation
	}
	_, err := svc.UpsertUserProfile(ctx, connect.NewRequest(&v1.UpsertUserProfileRequest{Profile: bad}))
	if codeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want CodeInvalidArgument, got %v", err)
	}
}

func TestUpsertUserProfile_Update(t *testing.T) {
	svc := newSvc(t)
	profile := validProfile("u1")
	if _, err := svc.UpsertUserProfile(ctx, connect.NewRequest(&v1.UpsertUserProfileRequest{Profile: profile})); err != nil {
		t.Fatal(err)
	}
	// Update the profile.
	updated := validProfile("u1")
	updated.CycleRegularity = v1.CycleRegularity_CYCLE_REGULARITY_SOMEWHAT_IRREGULAR
	resp, err := svc.UpsertUserProfile(ctx, connect.NewRequest(&v1.UpsertUserProfileRequest{Profile: updated}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetProfile().GetCycleRegularity() != v1.CycleRegularity_CYCLE_REGULARITY_SOMEWHAT_IRREGULAR {
		t.Error("profile was not updated")
	}
}

// ─── CreateBleedingObservation ────────────────────────────────────────────────

func TestCreateBleeding_HappyPath(t *testing.T) {
	svc := newSvc(t)
	obs := validBleeding("b1", "u1", "2026-01-15")
	resp, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Observation: obs}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetObservation().GetId() != "b1" {
		t.Errorf("got id %q, want b1", resp.Msg.GetObservation().GetId())
	}
}

func TestCreateBleeding_AutoID(t *testing.T) {
	svc := newSvc(t)
	// No ID provided – service should assign one.
	obs := validBleeding("", "u1", "2026-01-15")
	resp, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Observation: obs}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetObservation().GetId() == "" {
		t.Error("expected auto-assigned ID, got empty string")
	}
}

func TestCreateBleeding_NilObservation(t *testing.T) {
	svc := newSvc(t)
	_, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{}))
	if codeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want CodeInvalidArgument, got %v", err)
	}
}

func TestCreateBleeding_DuplicateID(t *testing.T) {
	svc := newSvc(t)
	obs := validBleeding("b1", "u1", "2026-01-15")
	if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Observation: obs})); err != nil {
		t.Fatal(err)
	}
	_, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Observation: obs}))
	if codeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("want CodeAlreadyExists, got %v", err)
	}
}

func TestCreateBleeding_TriggersRedetection(t *testing.T) {
	store := memory.New()
	svc := newSvcWithStore(t, store)
	// Log two bleeding episodes separated by 4+ days → two cycles expected.
	for _, obs := range []*v1.BleedingObservation{
		validBleeding("b1", "u1", "2026-01-01"),
		validBleeding("b2", "u1", "2026-01-02"),
		validBleeding("b3", "u1", "2026-01-30"),
	} {
		if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Observation: obs})); err != nil {
			t.Fatal(err)
		}
	}
	resp, err := svc.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetCycles()) != 2 {
		t.Errorf("want 2 cycles after re-detection, got %d", len(resp.Msg.GetCycles()))
	}
}

// ─── CreateSymptomObservation ─────────────────────────────────────────────────

func TestCreateSymptom_HappyPath(t *testing.T) {
	svc := newSvc(t)
	obs := validSymptom("s1", "u1", "2026-01-15")
	resp, err := svc.CreateSymptomObservation(ctx, connect.NewRequest(&v1.CreateSymptomObservationRequest{Observation: obs}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetObservation().GetId() != "s1" {
		t.Errorf("got id %q, want s1", resp.Msg.GetObservation().GetId())
	}
}

func TestCreateSymptom_NilObservation(t *testing.T) {
	svc := newSvc(t)
	_, err := svc.CreateSymptomObservation(ctx, connect.NewRequest(&v1.CreateSymptomObservationRequest{}))
	if codeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want CodeInvalidArgument, got %v", err)
	}
}

func TestCreateSymptom_AutoID(t *testing.T) {
	svc := newSvc(t)
	obs := validSymptom("", "u1", "2026-01-15")
	resp, err := svc.CreateSymptomObservation(ctx, connect.NewRequest(&v1.CreateSymptomObservationRequest{Observation: obs}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetObservation().GetId() == "" {
		t.Error("expected auto-assigned ID, got empty")
	}
}

// ─── CreateMoodObservation ────────────────────────────────────────────────────

func TestCreateMood_HappyPath(t *testing.T) {
	svc := newSvc(t)
	obs := validMood("m1", "u1", "2026-01-15")
	resp, err := svc.CreateMoodObservation(ctx, connect.NewRequest(&v1.CreateMoodObservationRequest{Observation: obs}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetObservation().GetId() != "m1" {
		t.Errorf("got id %q, want m1", resp.Msg.GetObservation().GetId())
	}
}

func TestCreateMood_NilObservation(t *testing.T) {
	svc := newSvc(t)
	_, err := svc.CreateMoodObservation(ctx, connect.NewRequest(&v1.CreateMoodObservationRequest{}))
	if codeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want CodeInvalidArgument, got %v", err)
	}
}

// ─── CreateMedication ─────────────────────────────────────────────────────────

func TestCreateMedication_HappyPath(t *testing.T) {
	svc := newSvc(t)
	med := validMedication("med1", "u1")
	resp, err := svc.CreateMedication(ctx, connect.NewRequest(&v1.CreateMedicationRequest{Medication: med}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetMedication().GetId() != "med1" {
		t.Errorf("got id %q, want med1", resp.Msg.GetMedication().GetId())
	}
}

func TestCreateMedication_NilMedication(t *testing.T) {
	svc := newSvc(t)
	_, err := svc.CreateMedication(ctx, connect.NewRequest(&v1.CreateMedicationRequest{}))
	if codeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want CodeInvalidArgument, got %v", err)
	}
}

func TestCreateMedication_AutoID(t *testing.T) {
	svc := newSvc(t)
	med := validMedication("", "u1")
	resp, err := svc.CreateMedication(ctx, connect.NewRequest(&v1.CreateMedicationRequest{Medication: med}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetMedication().GetId() == "" {
		t.Error("expected auto-assigned ID")
	}
}

// ─── CreateMedicationEvent ────────────────────────────────────────────────────

func TestCreateMedEvent_HappyPath(t *testing.T) {
	store := memory.New()
	// Pre-create the medication that the event references.
	if err := store.Medications().Create(ctx, validMedication("med1", "u1")); err != nil {
		t.Fatal(err)
	}
	svc := newSvcWithStore(t, store)
	ev := validMedEvent("ev1", "u1", "med1", "2026-01-15")
	resp, err := svc.CreateMedicationEvent(ctx, connect.NewRequest(&v1.CreateMedicationEventRequest{Event: ev}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetEvent().GetId() != "ev1" {
		t.Errorf("got id %q, want ev1", resp.Msg.GetEvent().GetId())
	}
}

func TestCreateMedEvent_MissingMedication(t *testing.T) {
	svc := newSvc(t)
	// medication "med1" does not exist in the store.
	ev := validMedEvent("ev1", "u1", "med1", "2026-01-15")
	_, err := svc.CreateMedicationEvent(ctx, connect.NewRequest(&v1.CreateMedicationEventRequest{Event: ev}))
	if codeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want CodeInvalidArgument for missing medication, got %v", err)
	}
}

func TestCreateMedEvent_NilEvent(t *testing.T) {
	svc := newSvc(t)
	_, err := svc.CreateMedicationEvent(ctx, connect.NewRequest(&v1.CreateMedicationEventRequest{}))
	if codeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want CodeInvalidArgument, got %v", err)
	}
}

// ─── ListCycles ───────────────────────────────────────────────────────────────

func TestListCycles_EmptyUser(t *testing.T) {
	svc := newSvc(t)
	resp, err := svc.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetCycles()) != 0 {
		t.Errorf("want 0 cycles, got %d", len(resp.Msg.GetCycles()))
	}
}

func TestListCycles_DetectsFromObs(t *testing.T) {
	svc := newSvc(t)
	// Log a single bleeding day.
	obs := validBleeding("b1", "u1", "2026-01-01")
	if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Observation: obs})); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetCycles()) != 1 {
		t.Fatalf("want 1 cycle, got %d", len(resp.Msg.GetCycles()))
	}
	if got := resp.Msg.GetCycles()[0].GetStartDate().GetValue(); got != "2026-01-01" {
		t.Errorf("start_date = %q, want 2026-01-01", got)
	}
}

// ─── ListTimeline ─────────────────────────────────────────────────────────────

func TestListTimeline_Empty(t *testing.T) {
	svc := newSvc(t)
	resp, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetRecords()) != 0 {
		t.Errorf("want 0 records, got %d", len(resp.Msg.GetRecords()))
	}
}

func TestListTimeline_MixedRecords_SortedDescending(t *testing.T) {
	svc := newSvc(t)
	// Add a bleeding obs and a symptom obs on different days.
	if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
		Observation: validBleeding("b1", "u1", "2026-01-10"),
	})); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSymptomObservation(ctx, connect.NewRequest(&v1.CreateSymptomObservationRequest{
		Observation: validSymptom("s1", "u1", "2026-01-15"),
	})); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	records := resp.Msg.GetRecords()
	// We expect at least 2 records: symptom + bleeding (plus any derived cycles).
	// The symptom (Jan 15) should come first (most recent).
	var foundSymptomFirst bool
	for i, r := range records {
		if r.GetSymptomObservation() != nil {
			foundSymptomFirst = i == 0
			break
		}
	}
	if !foundSymptomFirst {
		t.Error("symptom observation (Jan 15) should appear before bleeding (Jan 10) in descending order")
	}
}

func TestListTimeline_DateRangeFilter(t *testing.T) {
	svc := newSvc(t)
	if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
		Observation: validBleeding("b1", "u1", "2026-01-05"),
	})); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
		Observation: validBleeding("b2", "u1", "2026-01-20"),
	})); err != nil {
		t.Fatal(err)
	}
	// Request only Jan 15–31; should NOT include b1 (Jan 5).
	resp, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
		UserId: "u1",
		Range: &v1.DateRange{
			Start: &v1.LocalDate{Value: "2026-01-15"},
			End:   &v1.LocalDate{Value: "2026-01-31"},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range resp.Msg.GetRecords() {
		if bo := r.GetBleedingObservation(); bo != nil && bo.GetId() == "b1" {
			t.Error("b1 (Jan 5) should not appear in Jan 15–31 range")
		}
	}
}

func TestListTimeline_Pagination(t *testing.T) {
	svc := newSvc(t)
	// Create 5 bleeding observations on separate days.
	for i := 1; i <= 5; i++ {
		date := "2026-01-0" + string(rune('0'+i))
		obs := validBleeding("b"+string(rune('0'+i)), "u1", date)
		if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Observation: obs})); err != nil {
			t.Fatal(err)
		}
	}
	// Request page size 2, first page.
	resp1, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
		UserId:     "u1",
		Pagination: &v1.PaginationRequest{PageSize: 2},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp1.Msg.GetRecords()) != 2 {
		t.Errorf("page 1: want 2 records, got %d", len(resp1.Msg.GetRecords()))
	}
	nextToken := resp1.Msg.GetPagination().GetNextPageToken()
	if nextToken == "" {
		t.Fatal("expected a next_page_token for page 1")
	}
	// Fetch second page.
	resp2, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
		UserId:     "u1",
		Pagination: &v1.PaginationRequest{PageSize: 2, PageToken: nextToken},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp2.Msg.GetRecords()) != 2 {
		t.Errorf("page 2: want 2 records, got %d", len(resp2.Msg.GetRecords()))
	}
}

// ─── ListPredictions ──────────────────────────────────────────────────────────

func TestListPredictions_ReturnsEmpty(t *testing.T) {
	svc := newSvc(t)
	resp, err := svc.ListPredictions(ctx, connect.NewRequest(&v1.ListPredictionsRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetPredictions()) != 0 {
		t.Errorf("want 0 predictions (stub), got %d", len(resp.Msg.GetPredictions()))
	}
}

// ─── ListInsights ─────────────────────────────────────────────────────────────

func TestListInsights_ReturnsEmpty(t *testing.T) {
	svc := newSvc(t)
	resp, err := svc.ListInsights(ctx, connect.NewRequest(&v1.ListInsightsRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetInsights()) != 0 {
		t.Errorf("want 0 insights (stub), got %d", len(resp.Msg.GetInsights()))
	}
}

// ─── ExportData ───────────────────────────────────────────────────────────────

func TestExportData_EmptyUser(t *testing.T) {
	svc := newSvc(t)
	resp, err := svc.ExportData(ctx, connect.NewRequest(&v1.ExportDataRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetData()) == 0 {
		t.Error("expected non-empty export data even for empty user")
	}
	// Parse the JSON envelope.
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(resp.Msg.GetData(), &payload); err != nil {
		t.Fatalf("export data is not valid JSON: %v", err)
	}
}

func TestExportData_WithRecords(t *testing.T) {
	svc := newSvc(t)
	profile := validProfile("u1")
	if _, err := svc.UpsertUserProfile(ctx, connect.NewRequest(&v1.UpsertUserProfileRequest{Profile: profile})); err != nil {
		t.Fatal(err)
	}
	obs := validBleeding("b1", "u1", "2026-01-15")
	if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Observation: obs})); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.ExportData(ctx, connect.NewRequest(&v1.ExportDataRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	// Verify the JSON structure contains profile and bleeding_observations.
	var payload struct {
		Version     string            `json:"version"`
		UserID      string            `json:"user_id"`
		Profile     json.RawMessage   `json:"profile"`
		BleedingObs []json.RawMessage `json:"bleeding_observations"`
	}
	if err := json.Unmarshal(resp.Msg.GetData(), &payload); err != nil {
		t.Fatalf("unmarshal export: %v", err)
	}
	if payload.Version != "1" {
		t.Errorf("version = %q, want 1", payload.Version)
	}
	if payload.UserID != "u1" {
		t.Errorf("user_id = %q, want u1", payload.UserID)
	}
	if len(payload.Profile) == 0 {
		t.Error("expected profile in export")
	}
	if len(payload.BleedingObs) != 1 {
		t.Errorf("want 1 bleeding obs in export, got %d", len(payload.BleedingObs))
	}
}

// ─── ImportData ───────────────────────────────────────────────────────────────

func TestImportData_InvalidJSON(t *testing.T) {
	svc := newSvc(t)
	_, err := svc.ImportData(ctx, connect.NewRequest(&v1.ImportDataRequest{Data: []byte("not json")}))
	if codeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want CodeInvalidArgument, got %v", err)
	}
}

func TestImportData_WrongVersion(t *testing.T) {
	svc := newSvc(t)
	data, _ := json.Marshal(map[string]string{"version": "99", "user_id": "u1"})
	_, err := svc.ImportData(ctx, connect.NewRequest(&v1.ImportDataRequest{Data: data}))
	if codeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want CodeInvalidArgument for version mismatch, got %v", err)
	}
}

func TestImportData_RoundTrip(t *testing.T) {
	// Create a source service with some data.
	srcSvc := newSvc(t)
	profile := validProfile("u1")
	if _, err := srcSvc.UpsertUserProfile(ctx, connect.NewRequest(&v1.UpsertUserProfileRequest{Profile: profile})); err != nil {
		t.Fatal(err)
	}
	if _, err := srcSvc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
		Observation: validBleeding("b1", "u1", "2026-01-15"),
	})); err != nil {
		t.Fatal(err)
	}
	if _, err := srcSvc.CreateSymptomObservation(ctx, connect.NewRequest(&v1.CreateSymptomObservationRequest{
		Observation: validSymptom("s1", "u1", "2026-01-16"),
	})); err != nil {
		t.Fatal(err)
	}
	// Export from source.
	exportResp, err := srcSvc.ExportData(ctx, connect.NewRequest(&v1.ExportDataRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	// Import into a fresh service.
	dstSvc := newSvc(t)
	importResp, err := dstSvc.ImportData(ctx, connect.NewRequest(&v1.ImportDataRequest{Data: exportResp.Msg.GetData()}))
	if err != nil {
		t.Fatal(err)
	}
	// 1 profile + 1 bleeding + 1 symptom = 3 newly created records.
	if importResp.Msg.GetRecordsImported() != 3 {
		t.Errorf("want 3 records imported, got %d", importResp.Msg.GetRecordsImported())
	}
	// Verify the profile exists in destination.
	profResp, err := dstSvc.GetUserProfile(ctx, connect.NewRequest(&v1.GetUserProfileRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	if profResp.Msg.GetProfile().GetId() != "u1" {
		t.Error("profile not found after import")
	}
}

func TestImportData_IdempotentReImport(t *testing.T) {
	svc := newSvc(t)
	profile := validProfile("u1")
	if _, err := svc.UpsertUserProfile(ctx, connect.NewRequest(&v1.UpsertUserProfileRequest{Profile: profile})); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
		Observation: validBleeding("b1", "u1", "2026-01-15"),
	})); err != nil {
		t.Fatal(err)
	}
	exportResp, err := svc.ExportData(ctx, connect.NewRequest(&v1.ExportDataRequest{UserId: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	// Import into the SAME service twice; second import should find all
	// records already present and return 0 newly created.
	if _, err := svc.ImportData(ctx, connect.NewRequest(&v1.ImportDataRequest{Data: exportResp.Msg.GetData()})); err != nil {
		t.Fatal(err)
	}
	resp2, err := svc.ImportData(ctx, connect.NewRequest(&v1.ImportDataRequest{Data: exportResp.Msg.GetData()}))
	if err != nil {
		t.Fatal(err)
	}
	// Profile uses Upsert (always counts as 1), bleeding already exists (skipped).
	if resp2.Msg.GetRecordsImported() != 1 {
		t.Errorf("second import: want 1 (profile upsert), got %d", resp2.Msg.GetRecordsImported())
	}
}
