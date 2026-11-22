package activities_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/activities"
)

func TestSamplesFollowsCursorAndSendsServiceToken(t *testing.T) {
	id := uuid.New()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer s3cret", r.Header.Get("Authorization"))
		require.Equal(t, "/internal/atividades/"+id.String()+"/amostras", r.URL.Path)
		calls++
		if r.URL.Query().Get("cursor") == "" {
			fmt.Fprint(w, `{"amostras":[{"ts":"2026-03-01T07:00:00Z","fc_bpm":120,"potencia_w":200}],"proximo_cursor":"c1"}`)
			return
		}
		require.Equal(t, "c1", r.URL.Query().Get("cursor"))
		fmt.Fprint(w, `{"amostras":[{"ts":"2026-03-01T07:00:01Z","velocidade":3.5}],"proximo_cursor":null}`)
	}))
	defer srv.Close()

	got, err := activities.NewClient(srv.URL, "s3cret").Samples(t.Context(), id)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, 2, calls)
	require.Equal(t, 120, *got[0].HR)
	require.InDelta(t, 3.5, *got[1].Speed, 1e-9)
}

func TestSamplesReportsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	_, err := activities.NewClient(srv.URL, "x").Samples(t.Context(), uuid.New())
	require.ErrorIs(t, err, activities.ErrNotFound)
}

// O provedor (services/ingest) valida as chaves da resposta contra o mesmo arquivo.
func TestSamplesDecodesContractExample(t *testing.T) {
	raw, err := os.ReadFile("../../../../../contracts/openapi/examples/atividades-amostras-pagina.json")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "" {
			_, _ = w.Write(raw)
			return
		}
		fmt.Fprint(w, `{"amostras":[],"proximo_cursor":null}`)
	}))
	defer srv.Close()

	got, err := activities.NewClient(srv.URL, "x").Samples(t.Context(), uuid.New())
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, 120, *got[0].HR)
	require.Equal(t, 200, *got[0].Power)
	require.InDelta(t, 8.3, *got[0].Speed, 1e-9)
	require.InDelta(t, 12.5, *got[0].Altitude, 1e-9)
	require.Nil(t, got[1].HR)
}
