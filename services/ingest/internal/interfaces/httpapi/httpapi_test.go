package httpapi_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
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
