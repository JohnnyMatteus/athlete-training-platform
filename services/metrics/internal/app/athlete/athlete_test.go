package athlete_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/athlete"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

type fakeStore struct {
	athlete.Store
	links     map[[2]uuid.UUID][]string
	thresh    []athlete.Threshold
	events    []athlete.OutEvent
	loadOwner uuid.UUID
}

func (f *fakeStore) HasActiveLink(_ context.Context, a, c uuid.UUID, cat string) (bool, error) {
	for _, x := range f.links[[2]uuid.UUID{a, c}] {
		if x == cat {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeStore) DailyLoad(_ context.Context, a uuid.UUID, _, _ load.Day) ([]athlete.DayLoad, error) {
	f.loadOwner = a
	return nil, nil
}
func (f *fakeStore) SaveThreshold(_ context.Context, _ uuid.UUID, t athlete.Threshold, ev athlete.OutEvent) error {
	f.thresh = append(f.thresh, t)
	f.events = append(f.events, ev)
	return nil
}

func TestCoachReadsOnlyWithActiveLinkAndCategory(t *testing.T) {
	ath, coach := uuid.New(), uuid.New()
	st := &fakeStore{links: map[[2]uuid.UUID][]string{{ath, coach}: {athlete.CategoryTraining}}}
	svc := athlete.NewService(st, time.Now)
	p := athlete.Principal{ID: coach, Roles: []string{athlete.RoleCoach}}

	_, _, err := svc.Load(t.Context(), p, &ath, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, ath, st.loadOwner)

	_, err = svc.Profile(t.Context(), p, &ath) // categoria frequencia_cardiaca não consentida
	assert.ErrorIs(t, err, athlete.ErrNotFound)

	stranger := athlete.Principal{ID: uuid.New(), Roles: []string{athlete.RoleCoach}}
	_, _, err = svc.Load(t.Context(), stranger, &ath, nil, nil)
	assert.ErrorIs(t, err, athlete.ErrNotFound)

	other := athlete.Principal{ID: uuid.New(), Roles: []string{athlete.RoleAthlete}}
	_, _, err = svc.Load(t.Context(), other, &ath, nil, nil)
	assert.ErrorIs(t, err, athlete.ErrNotFound, "atleta comum não lê outro atleta")
}

func TestCoachNeverWritesAndThresholdPublishesEvent(t *testing.T) {
	st := &fakeStore{}
	svc := athlete.NewService(st, time.Now)
	coach := athlete.Principal{ID: uuid.New(), Roles: []string{athlete.RoleCoach}}
	th := athlete.Threshold{Kind: "ftp", Value: 250, ValidFrom: "2026-11-01"}
	assert.ErrorIs(t, svc.PutThreshold(t.Context(), coach, th), athlete.ErrForbidden)
	assert.ErrorIs(t, svc.PutProfile(t.Context(), coach, athlete.Profile{Sex: "M", RestingHR: 50, MaxHR: 190}), athlete.ErrForbidden)

	me := athlete.Principal{ID: uuid.New(), Roles: []string{athlete.RoleAthlete}}
	require.NoError(t, svc.PutThreshold(t.Context(), me, th))
	require.Len(t, st.events, 1)
	assert.Equal(t, "limiar.atualizado", st.events[0].RoutingKey)

	assert.ErrorIs(t, svc.PutThreshold(t.Context(), me, athlete.Threshold{Kind: "ftp", Value: -1, ValidFrom: "2026-11-01"}), athlete.ErrInvalid)
	assert.ErrorIs(t, svc.PutProfile(t.Context(), me, athlete.Profile{Sex: "M", RestingHR: 90, MaxHR: 80}), athlete.ErrInvalid)
}
