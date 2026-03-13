package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/oklog/ulid/v2"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	"github.com/2ajoyce/openmenses/engine/internal/validation"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// newID returns a fresh ULID string for use as a record identifier.
func newID() string {
	return ulid.Make().String()
}

// toConnectErr maps internal errors to Connect-RPC error codes.
func toConnectErr(err error) error {
	if err == nil {
		return nil
	}
	var valErr *validation.Error
	if errors.As(err, &valErr) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, storage.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, storage.ErrConflict) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	if errors.Is(err, storage.ErrInvalidInput) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

// pageReq converts a proto PaginationRequest to a storage PageRequest.
func pageReq(p *v1.PaginationRequest) storage.PageRequest {
	if p == nil {
		return storage.PageRequest{}
	}
	return storage.PageRequest{PageSize: p.GetPageSize(), PageToken: p.GetPageToken()}
}

// paginateAll calls fetch repeatedly with successive page tokens until all
// pages have been collected and returns the combined slice.
func paginateAll[T any](ctx context.Context, fetch func(ctx context.Context, token string) ([]T, string, error)) ([]T, error) {
	var all []T
	token := ""
	for {
		items, next, err := fetch(ctx, token)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if next == "" {
			break
		}
		token = next
	}
	return all, nil
}

var protoJSONMarshal = protojson.MarshalOptions{EmitUnpopulated: false}

// marshalProtoJSON serialises a proto message to a protojson-encoded
// json.RawMessage.
func marshalProtoJSON(msg proto.Message) (json.RawMessage, error) {
	b, err := protoJSONMarshal.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// marshalAll serialises a slice of proto messages to a slice of
// json.RawMessage values.
func marshalAll[T proto.Message](items []T) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		b, err := marshalProtoJSON(item)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// applyUserProfileFieldMask merges updates into existing according to the
// field paths specified in mask. If mask is nil or empty, all fields from
// updates are used (full replace). Returns an error if any path in the mask
// is unrecognized.
func applyUserProfileFieldMask(existing, updates *v1.UserProfile, mask *fieldmaskpb.FieldMask) (*v1.UserProfile, error) {
	// If no mask or empty mask, do a full replace of all fields
	if mask == nil || len(mask.GetPaths()) == 0 {
		return &v1.UserProfile{
			Name:             updates.GetName(),
			BiologicalCycle:  updates.GetBiologicalCycle(),
			Contraception:    updates.GetContraception(),
			CycleRegularity:  updates.GetCycleRegularity(),
			ReproductiveGoal: updates.GetReproductiveGoal(),
			HealthConditions: updates.GetHealthConditions(),
			TrackingFocus:    updates.GetTrackingFocus(),
		}, nil
	}

	// Apply only the fields specified in the mask.
	// Note: "name" is intentionally excluded to prevent changing the profile ID.
	for _, path := range mask.GetPaths() {
		switch path {
		case "biological_cycle":
			existing.BiologicalCycle = updates.BiologicalCycle
		case "contraception":
			existing.Contraception = updates.Contraception
		case "cycle_regularity":
			existing.CycleRegularity = updates.CycleRegularity
		case "reproductive_goal":
			existing.ReproductiveGoal = updates.ReproductiveGoal
		case "health_conditions":
			existing.HealthConditions = updates.HealthConditions
		case "tracking_focus":
			existing.TrackingFocus = updates.TrackingFocus
		default:
			return nil, fmt.Errorf("invalid field mask path: %q", path)
		}
	}
	return existing, nil
}
