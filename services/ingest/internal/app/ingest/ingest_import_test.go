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

func setup(tr port.Translator, fileType string, cfg ingest.Config) (*memDB, *ingest.IngestImport) {
	db := newMemDB()
	return db, ingest.New(fakeObjects{}, db, map[string]port.Translator{fileType: tr}, cfg, nil)
}

func recv(fileType string) port.ImportReceived {
	return port.ImportReceived{EventID: uuid.New(), ImportID: uuid.New(), AthleteID: uuid.New(), FileType: fileType,
		Bucket: "uploads", ObjectKey: "abc", Attempt: 1}
}

func done(db *memDB, i int) port.ImportDone {
	return db.eventsOf(port.EventImportDone)[i].Payload.(port.ImportDone)
}

func TestNewImport(t *testing.T) {
	db, uc := setup(three(), "zip", ingest.Config{})
	require.NoError(t, uc.Handle(context.Background(), recv("zip")))

	assert.Len(t, db.acts, 3)
	assert.Equal(t, 350, sum(db.samples))
	imported := db.eventsOf(port.EventActivityImported)
	assert.Len(t, imported, 3)
	assert.Equal(t, 100, imported[0].Payload.(port.ActivityImported).Samples)
	require.Len(t, db.eventsOf(port.EventImportDone), 1)
	assert.Equal(t, 3, done(db, 0).Imported)
	assert.Equal(t, 0, done(db, 0).Duplicated)
}

func TestReimportIsIdempotent(t *testing.T) {
	db, uc := setup(three(), "zip", ingest.Config{})
	require.NoError(t, uc.Handle(context.Background(), recv("zip")))
	require.NoError(t, uc.Handle(context.Background(), recv("zip"))) // outro lote, mesmo arquivo

	assert.Len(t, db.acts, 3)
	assert.Equal(t, 350, sum(db.samples))
	assert.Len(t, db.eventsOf(port.EventActivityImported), 3, "sem novos TreinoImportado")
	assert.Equal(t, 3, done(db, 1).Duplicated)
	assert.Equal(t, 0, done(db, 1).Imported)
}

func TestSameMessageDeliveredTwice(t *testing.T) {
	db, uc := setup(three(), "zip", ingest.Config{})
	msg := recv("zip")
	require.NoError(t, uc.Handle(context.Background(), msg))
	require.NoError(t, uc.Handle(context.Background(), msg))
	assert.Len(t, db.eventsOf(port.EventImportDone), 1)
	assert.Len(t, db.acts, 3)
}

func TestFailureRetriesThenFails(t *testing.T) {
	db, uc := setup(three(), "zip", ingest.Config{MaxAttempts: 2})
	db.failNext = errors.New("banco indisponível")
	msg := recv("zip")

	require.Error(t, uc.Handle(context.Background(), msg), "1ª tentativa: nack")
	assert.Empty(t, db.acts, "rollback")
	assert.Empty(t, db.eventsOf(port.EventImportFailed))

	msg.Attempt = 2
	require.NoError(t, uc.Handle(context.Background(), msg), "limite esgotado: ack")
	failed := db.eventsOf(port.EventImportFailed)
	require.Len(t, failed, 1)
	f := failed[0].Payload.(port.ImportFailed)
	assert.Equal(t, 2, f.Attempts)
	assert.Equal(t, port.CodeInternal, f.Code)
	assert.Empty(t, db.eventsOf(port.EventImportDone))
}

func TestMalformedFileFailsWithoutRetry(t *testing.T) {
	tr := three()
	tr.err = port.ErrMalformed
	db, uc := setup(tr, "zip", ingest.Config{MaxAttempts: 5})
	require.NoError(t, uc.Handle(context.Background(), recv("zip")))
	assert.Empty(t, db.acts)
	assert.Equal(t, port.CodeInvalidFile, db.eventsOf(port.EventImportFailed)[0].Payload.(port.ImportFailed).Code)
}

func TestUnsupportedType(t *testing.T) {
	db, uc := setup(three(), "zip", ingest.Config{})
	require.NoError(t, uc.Handle(context.Background(), recv("xyz")))
	assert.Equal(t, port.CodeUnsupportedFormat, db.eventsOf(port.EventImportFailed)[0].Payload.(port.ImportFailed).Code)
}

func TestSamplesWrittenInBatches(t *testing.T) {
	tr := &scriptTranslator{items: []scripted{{act(activity.SourceFit, activity.TypeRunning, 0, 10000, activity.ChanHeartRate), 10000}}}
	db, uc := setup(tr, "fit", ingest.Config{BatchSize: 1000})
	require.NoError(t, uc.Handle(context.Background(), recv("fit")))

	assert.Equal(t, 10000, sum(db.samples))
	assert.Len(t, db.copies, 10)
	for _, n := range db.copies {
		assert.LessOrEqual(t, n, 1000)
	}
}

func TestCrossSourceDedupAnyOrder(t *testing.T) {
	watch := &scriptTranslator{items: []scripted{{act(activity.SourceAppleExport, activity.Cycling(activity.SettingOutdoor), 0, 3600, activity.ChanHeartRate), 30}}}
	bike := &scriptTranslator{items: []scripted{{act(activity.SourceFit, activity.Cycling(activity.SettingOutdoor), 2*time.Minute, 3500, activity.ChanHeartRate|activity.ChanPower), 40}}}

	for name, order := range map[string][]string{"apple_primeiro": {"zip", "fit"}, "fit_primeiro": {"fit", "zip"}} {
		t.Run(name, func(t *testing.T) {
			db := newMemDB()
			uc := ingest.New(fakeObjects{}, db, map[string]port.Translator{"zip": watch, "fit": bike}, ingest.Config{}, nil)
			athlete := uuid.New()
			for _, ft := range order {
				m := recv(ft)
				m.AthleteID = athlete
				require.NoError(t, uc.Handle(context.Background(), m))
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
