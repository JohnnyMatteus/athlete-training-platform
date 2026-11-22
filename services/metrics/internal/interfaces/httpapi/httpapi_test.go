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

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/jwtauth"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/athlete"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/interfaces/httpapi"
)

type store struct {
	athlete.Store
	link bool
}

func (s store) HasActiveLink(context.Context, uuid.UUID, uuid.UUID, string) (bool, error) {
	return s.link, nil
}
func (s store) DailyLoad(context.Context, uuid.UUID, load.Day, load.Day) ([]athlete.DayLoad, error) {
	return []athlete.DayLoad{{Day: "2026-03-01", TSS: 64, CTL: 1.5, ATL: 9, TSB: -7}}, nil
}
func (s store) SaveProfile(context.Context, uuid.UUID, athlete.Profile) error { return nil }

func setup(t *testing.T, link bool) (http.Handler, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	auth, err := jwtauth.New(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), "iss", "aud")
	require.NoError(t, err)
	return httpapi.New(athlete.NewService(store{link: link}, time.Now), auth).Handler(), key
}

func token(t *testing.T, key *rsa.PrivateKey, sub uuid.UUID, roles ...string) string {
	t.Helper()
	c := jwt.MapClaims{"iss": "iss", "aud": "aud", "sub": sub.String(), "exp": time.Now().Add(time.Hour).Unix()}
	if len(roles) > 0 {
		c["roles"] = roles
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodRS256, c).SignedString(key)
	require.NoError(t, err)
	return s
}

func do(h http.Handler, method, url, tok, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, url, strings.NewReader(body))
	if tok != "" {
		r.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestRequiresValidToken(t *testing.T) {
	h, key := setup(t, false)
	assert.Equal(t, 401, do(h, "GET", "/metricas/carga", "", "").Code)
	assert.Equal(t, 401, do(h, "GET", "/metricas/carga", "lixo", "").Code)
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	assert.Equal(t, 401, do(h, "GET", "/metricas/carga", token(t, other, uuid.New()), "").Code, "assinatura de outra chave")
	assert.Equal(t, 200, do(h, "GET", "/metricas/carga", token(t, key, uuid.New()), "").Code)
}

func TestCoachNeedsLinkAndCannotWrite(t *testing.T) {
	ath := uuid.New()
	coach := uuid.New()
	h, key := setup(t, false)
	assert.Equal(t, 404, do(h, "GET", "/metricas/carga?atleta_id="+ath.String(), token(t, key, coach, "treinador"), "").Code)
	assert.Equal(t, 403, do(h, "PUT", "/metricas/perfil", token(t, key, coach, "treinador"), `{"sexo":"M","fc_repouso":50,"fc_max":190}`).Code)

	h, key = setup(t, true)
	assert.Equal(t, 200, do(h, "GET", "/metricas/carga?atleta_id="+ath.String(), token(t, key, coach, "treinador"), "").Code)
	assert.Equal(t, 422, do(h, "GET", "/metricas/carga?de=ontem", token(t, key, coach, "treinador"), "").Code)
}
