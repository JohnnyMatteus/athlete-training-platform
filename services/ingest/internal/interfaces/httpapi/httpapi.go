// Package httpapi expõe contracts/openapi/atividades.yaml: amostras (token de serviço) e correção (JWT do atleta).
package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/activities"
)

// AthleteAuth valida o token do atleta e devolve o atleta_id.
type AthleteAuth interface {
	Athlete(token string) (uuid.UUID, error)
}

type API struct {
	samples      activities.SampleReader
	corrector    *activities.Corrector
	auth         AthleteAuth
	serviceToken []byte
}

func New(samples activities.SampleReader, c *activities.Corrector, auth AthleteAuth, serviceToken string) *API {
	return &API{samples: samples, corrector: c, auth: auth, serviceToken: []byte(serviceToken)}
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /internal/atividades/{id}/amostras", a.listSamples)
	mux.HandleFunc("PATCH /atividades/{id}", a.correct)
	return otelhttp.NewHandler(mux, "ingest-http")
}

func bearer(r *http.Request) string {
	t, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return t
}

func (a *API) listSamples(w http.ResponseWriter, r *http.Request) {
	// Sem token configurado o endpoint fica fechado: comparação em tempo constante.
	if len(a.serviceToken) == 0 || subtle.ConstantTimeCompare([]byte(bearer(r)), a.serviceToken) != 1 {
		problem(w, http.StatusUnauthorized, "Não autenticado")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		problem(w, http.StatusNotFound, "Atividade inexistente")
		return
	}
	limit := 0
	if v := r.URL.Query().Get("limite"); v != "" {
		if limit, err = strconv.Atoi(v); err != nil || limit < 1 {
			problem(w, http.StatusUnprocessableEntity, "limite inválido")
			return
		}
	}
	p, err := activities.Samples(r.Context(), a.samples, id, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		fail(w, err)
		return
	}
	items := make([]map[string]any, 0, len(p.Samples))
	for _, s := range p.Samples {
		items = append(items, map[string]any{
			"ts": s.TS.UTC(), "fc_bpm": s.HR, "potencia_w": s.Power, "cadencia": s.Cadence,
			"velocidade": s.Speed, "altitude_m": s.Altitude,
		})
	}
	var next *string
	if p.Next != "" {
		next = &p.Next
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"atividade_id": p.Header.ActivityID, "atleta_id": p.Header.AthleteID, "tipo": p.Header.Type,
		"inicio_utc": p.Header.Start.UTC(), "duracao_s": p.Header.DurationS, "amostras": items, "proximo_cursor": next,
	})
}

type patchBody struct {
	Type  *string    `json:"tipo"`
	Start *time.Time `json:"inicio_utc"`
}

func (a *API) correct(w http.ResponseWriter, r *http.Request) {
	athlete, err := a.auth.Athlete(bearer(r))
	if err != nil {
		problem(w, http.StatusUnauthorized, "Não autenticado")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		problem(w, http.StatusNotFound, "Atividade inexistente")
		return
	}
	var b patchBody
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		problem(w, http.StatusUnprocessableEntity, "corpo inválido")
		return
	}
	res, err := a.corrector.Correct(r.Context(), athlete, id, activities.Patch{Type: b.Type, Start: b.Start})
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"atividade_id": res.ID, "tipo": res.Type, "inicio_utc": res.Start, "duracao_s": res.DurationS, "distancia_m": res.DistanceM,
	})
}

func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, activities.ErrNotFound):
		problem(w, http.StatusNotFound, "Atividade inexistente")
	case errors.Is(err, activities.ErrConflict):
		problem(w, http.StatusConflict, "Correção colide com outra atividade")
	case errors.Is(err, activities.ErrInvalid):
		problem(w, http.StatusUnprocessableEntity, err.Error())
	default:
		problem(w, http.StatusInternalServerError, "Erro interno")
	}
}

func problem(w http.ResponseWriter, status int, title string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": title, "status": status})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
