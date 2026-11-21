package ingest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/ingest"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

var base = time.Date(2024, 3, 10, 10, 0, 0, 0, time.UTC)

func act(src activity.Source, ty activity.Type, off time.Duration, durS int, ch activity.Channels) func(uuid.UUID) *activity.Activity {
	return func(athlete uuid.UUID) *activity.Activity {
		i, _ := activity.NewInstant(base.Add(off))
		a, err := activity.New(activity.Params{AthleteID: athlete, Type: ty, Source: src, Device: "d-" + string(src), Start: i, DurationS: durS, Channels: ch})
		if err != nil {
			panic(err)
		}
		return a
	}
}

func three() *scriptTranslator {
	return &scriptTranslator{items: []scripted{
		{act(activity.SourceAppleExport, activity.TypeRunning, 0, 1800, activity.ChanHeartRate), 100},
		{act(activity.SourceAppleExport, activity.Cycling(activity.SettingOutdoor), 24*time.Hour, 3600, activity.ChanHeartRate|activity.ChanPower), 200},
		{act(activity.SourceAppleExport, activity.Swim(activity.SettingPool), 48*time.Hour, 2400, activity.ChanHeartRate), 50},
	}}
}

func setup(tr port.Translator, format activity.Source, cfg ingest.Config) (*memDB, *ingest.IngestImport, port.Import) {
	db := newMemDB()
	uc := ingest.New(db, fakeObjects{}, db, map[activity.Source]port.Translator{format: tr}, cfg, nil)
	return db, uc, db.addImport(format)
}

func recv(imp port.Import) port.ImportReceived {
	return port.ImportReceived{EventID: uuid.New(), ImportID: imp.ID}
}

func TestNewImport(t *testing.T) {
	db, uc, imp := setup(three(), activity.SourceAppleExport, ingest.Config{})
	require.NoError(t, uc.Handle(context.Background(), recv(imp)))

	assert.Len(t, db.acts, 3)
	assert.Equal(t, 350, sum(db.samples))
	assert.Len(t, db.eventsOf(port.EventActivityImport), 3)
	done := db.eventsOf(port.EventImportDone)
	require.Len(t, done, 1)
	assert.Equal(t, 3, done[0].Payload.(port.ImportDone).Imported)
	assert.Equal(t, 0, done[0].Payload.(port.ImportDone).Duplicated)
}

func TestReimportIsIdempotent(t *testing.T) {
	db, uc, imp := setup(three(), activity.SourceAppleExport, ingest.Config{})
	require.NoError(t, uc.Handle(context.Background(), recv(imp)))

	second := db.addImport(activity.SourceAppleExport)
	require.NoError(t, uc.Handle(context.Background(), recv(second)))

	assert.Len(t, db.acts, 3)
	assert.Equal(t, 350, sum(db.samples))
	assert.Len(t, db.eventsOf(port.EventActivityImport), 3, "sem novos TreinoImportado")
	assert.Equal(t, 3, db.totals[second.ID].Duplicated)
	assert.Equal(t, 0, db.totals[second.ID].Imported)
}

func TestSameMessageDeliveredTwice(t *testing.T) {
	db, uc, imp := setup(three(), activity.SourceAppleExport, ingest.Config{})
	msg := recv(imp)
	require.NoError(t, uc.Handle(context.Background(), msg))
	require.NoError(t, uc.Handle(context.Background(), msg))
	assert.Len(t, db.eventsOf(port.EventImportDone), 1)
	assert.Len(t, db.acts, 3)
}

func TestFailureRetriesThenFails(t *testing.T) {
	db, uc, imp := setup(three(), activity.SourceAppleExport, ingest.Config{MaxAttempts: 2})
	db.failNext = errors.New("banco indisponível")
	msg := recv(imp)

	require.Error(t, uc.Handle(context.Background(), msg), "1ª tentativa: nack")
	assert.Empty(t, db.acts, "rollback")
	assert.Empty(t, db.eventsOf(port.EventImportFailed))

	require.NoError(t, uc.Handle(context.Background(), msg), "2ª tentativa esgota o limite: ack")
	assert.Equal(t, "falhou", db.status[imp.ID])
	failed := db.eventsOf(port.EventImportFailed)
	require.Len(t, failed, 1)
	assert.Equal(t, 2, failed[0].Payload.(port.ImportFailed).Attempts)
	assert.Empty(t, db.eventsOf(port.EventImportDone))
}

func TestMalformedFileFailsWithoutRetry(t *testing.T) {
	tr := three()
	tr.err = port.ErrMalformed
	db, uc, imp := setup(tr, activity.SourceAppleExport, ingest.Config{MaxAttempts: 5})
	require.NoError(t, uc.Handle(context.Background(), recv(imp)))
	assert.Equal(t, "falhou", db.status[imp.ID])
	assert.Empty(t, db.acts)
}

func TestSamplesWrittenInBatches(t *testing.T) {
	tr := &scriptTranslator{items: []scripted{{act(activity.SourceFit, activity.TypeRunning, 0, 10000, activity.ChanHeartRate), 10000}}}
	db, uc, imp := setup(tr, activity.SourceFit, ingest.Config{BatchSize: 1000})
	require.NoError(t, uc.Handle(context.Background(), recv(imp)))

	assert.Equal(t, 10000, sum(db.samples))
	assert.Len(t, db.copies, 10)
	for _, n := range db.copies {
		assert.LessOrEqual(t, n, 1000)
	}
}

func TestCrossSourceDedupAnyOrder(t *testing.T) {
	watch := &scriptTranslator{items: []scripted{{act(activity.SourceAppleExport, activity.Cycling(activity.SettingOutdoor), 0, 3600, activity.ChanHeartRate), 30}}}
	bike := &scriptTranslator{items: []scripted{{act(activity.SourceFit, activity.Cycling(activity.SettingOutdoor), 2*time.Minute, 3500, activity.ChanHeartRate|activity.ChanPower), 40}}}

	type step struct {
		format activity.Source
		tr     *scriptTranslator
	}
	apple, fit := step{activity.SourceAppleExport, watch}, step{activity.SourceFit, bike}
	for name, order := range map[string][]step{"apple_primeiro": {apple, fit}, "fit_primeiro": {fit, apple}} {
		t.Run(name, func(t *testing.T) {
			db := newMemDB()
			translators := map[activity.Source]port.Translator{}
			var imports []port.Import
			for _, st := range order {
				translators[st.format] = st.tr
				imports = append(imports, db.addImport(st.format))
			}
			uc := ingest.New(db, fakeObjects{}, db, translators, ingest.Config{}, nil)
			for _, imp := range imports {
				require.NoError(t, uc.Handle(context.Background(), recv(imp)))
			}

			require.Len(t, db.acts, 1)
			for _, r := range db.acts {
				assert.Equal(t, activity.SourceFit, r.cand.Source)
				assert.Len(t, r.alts, 1)
			}
			assert.Equal(t, 40, sum(db.samples), "só as amostras da fonte principal")
		})
	}
}

func sum(m map[activity.ID]int) (n int) {
	for _, v := range m {
		n += v
	}
	return
}
