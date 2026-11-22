package httpapi_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/adapters/jwtauth"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/activities"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/interfaces/httpapi"
)

type reader struct{}

func (reader) Page(context.Context, uuid.UUID, *time.Time, int) (activities.Header, []activities.Sample, error) {
	return activities.Header{}, nil, nil
}

type uow struct{ owner uuid.UUID }

func (u uow) Do(ctx context.Context, fn func(context.Context, activities.CorrectionTx) error) error {
	return fn(ctx, u)
}
func (u uow) Lock(context.Context, uuid.UUID) (activities.Current, error) {
	return activities.Current{AthleteID: u.owner, Source: "fit", Type: "corrida", Start: time.Now()}, nil
}
func (uow) Update(context.Context, uuid.UUID, string, time.Time, string) error { return nil }
func (uow) Enqueue(context.Context, activities.OutEvent) error                 { return nil }

func setup(t *testing.T, owner uuid.UUID) (http.Handler, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	auth, err := jwtauth.New(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), "iss", "aud")
	require.NoError(t, err)
	return httpapi.New(reader{}, activities.NewCorrector(uow{owner}, time.Now), auth, "svc-token").Handler(), key
}

func token(t *testing.T, key *rsa.PrivateKey, sub uuid.UUID) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": "iss", "aud": "aud", "sub": sub.String(), "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(key)
	require.NoError(t, err)
	return s
}

func do(h http.Handler, method, url, tok, body string) int {
	r := httptest.NewRequest(method, url, strings.NewReader(body))
	if tok != "" {
		r.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code
}

func TestSamplesNeedServiceToken(t *testing.T) {
	h, _ := setup(t, uuid.New())
	url := "/internal/atividades/" + uuid.NewString() + "/amostras"
	assert.Equal(t, 401, do(h, "GET", url, "", ""))
	assert.Equal(t, 401, do(h, "GET", url, "errado", ""))
	assert.Equal(t, 200, do(h, "GET", url, "svc-token", ""))
}

func TestPatchOnlyByOwnerWithValidJWT(t *testing.T) {
	owner := uuid.New()
	h, key := setup(t, owner)
	url := "/atividades/" + uuid.NewString()
	body := `{"tipo":"forca"}`
	assert.Equal(t, 401, do(h, "PATCH", url, "", body))
	assert.Equal(t, 401, do(h, "PATCH", url, "svc-token", body), "token de serviço não vale como atleta")
	assert.Equal(t, 404, do(h, "PATCH", url, token(t, key, uuid.New()), body), "atividade de outro atleta")
	assert.Equal(t, 200, do(h, "PATCH", url, token(t, key, owner), body))
	assert.Equal(t, 422, do(h, "PATCH", url, token(t, key, owner), `{}`))
}

type pagedReader struct{}

func (pagedReader) Page(context.Context, uuid.UUID, *time.Time, int) (activities.Header, []activities.Sample, error) {
	i16, f32 := func(v int16) *int16 { return &v }, func(v float32) *float32 { return &v }
	t0 := time.Date(2026, 3, 1, 7, 0, 0, 0, time.UTC)
	return activities.Header{ActivityID: uuid.New(), AthleteID: uuid.New(), Type: "ciclismo_outdoor", Start: t0, DurationS: 3600},
		[]activities.Sample{
			{TS: t0, HR: i16(120), Power: i16(200), Cadence: i16(85), Speed: f32(8.3), Altitude: f32(12.5)},
			{TS: t0.Add(time.Second)},
			{TS: t0.Add(2 * time.Second)},
		}, nil
}

// O exemplo em contracts/ é lido também pelo cliente do metrics: se as chaves divergirem, um dos dois testes quebra.
func TestSamplesResponseMatchesContractExample(t *testing.T) {
	raw, err := os.ReadFile("../../../../../contracts/openapi/examples/atividades-amostras-pagina.json")
	require.NoError(t, err)
	var want map[string]any
	require.NoError(t, json.Unmarshal(raw, &want))

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	auth, err := jwtauth.New(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), "iss", "aud")
	require.NoError(t, err)
	h := httpapi.New(pagedReader{}, activities.NewCorrector(uow{uuid.New()}, time.Now), auth, "svc-token").Handler()

	r := httptest.NewRequest("GET", "/internal/atividades/"+uuid.NewString()+"/amostras?limite=2", nil)
	r.Header.Set("Authorization", "Bearer svc-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, 200, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))

	keys := func(m map[string]any) []string {
		out := make([]string, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	assert.Equal(t, keys(want), keys(got))
	assert.Equal(t, keys(want["amostras"].([]any)[0].(map[string]any)), keys(got["amostras"].([]any)[0].(map[string]any)))
	assert.NotNil(t, got["proximo_cursor"])
}
