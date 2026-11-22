// Package httpapi expõe a API de leitura e escrita descrita em contracts/openapi/metricas.yaml.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/jwtauth"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/athlete"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

// Authenticator é a porta de autenticação.
type Authenticator interface {
	Authenticate(token string) (athlete.Principal, error)
}

type API struct {
	svc  *athlete.Service
	auth Authenticator
}

func New(svc *athlete.Service, auth Authenticator) *API { return &API{svc: svc, auth: auth} }

type ctxKey struct{}

// Handler devolve o roteador com tracing HTTP.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /metricas/carga", a.authed(a.getLoad))
	mux.Handle("GET /metricas/atividades", a.authed(a.getActivities))
	mux.Handle("GET /metricas/limiares", a.authed(a.getThresholds))
	mux.Handle("PUT /metricas/limiares", a.authed(a.putThreshold))
	mux.Handle("GET /metricas/perfil", a.authed(a.getProfile))
	mux.Handle("PUT /metricas/perfil", a.authed(a.putProfile))
	return otelhttp.NewHandler(mux, "metrics-http")
}

func (a *API) authed(h func(http.ResponseWriter, *http.Request, athlete.Principal)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			problem(w, http.StatusUnauthorized, "Não autenticado")
			return
		}
		p, err := a.auth.Authenticate(token)
		if err != nil {
			problem(w, http.StatusUnauthorized, "Não autenticado")
			return
		}
		h(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, p)), p)
	})
}

func athleteParam(r *http.Request) (*uuid.UUID, error) {
	v := r.URL.Query().Get("atleta_id")
	if v == "" {
		return nil, nil
	}
	id, err := uuid.Parse(v)
	return &id, err
}

func dayParam(r *http.Request, name string) (*load.Day, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return nil, nil
	}
	d, err := load.ParseDay(v)
	return &d, err
}

type window struct {
	athlete  *uuid.UUID
	from, to *load.Day
}

func parseWindow(r *http.Request) (window, error) {
	var w window
	var err error
	if w.athlete, err = athleteParam(r); err != nil {
		return w, errors.New("atleta_id inválido")
	}
	if w.from, err = dayParam(r, "de"); err != nil {
		return w, errors.New("de inválido")
	}
	if w.to, err = dayParam(r, "ate"); err != nil {
		return w, errors.New("ate inválido")
	}
	return w, nil
}

func (a *API) getLoad(w http.ResponseWriter, r *http.Request, p athlete.Principal) {
	q, err := parseWindow(r)
	if err != nil {
		problem(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	id, rows, err := a.svc.Load(r.Context(), p, q.athlete, q.from, q.to)
	if err != nil {
		fail(w, err)
		return
	}
	days := make([]map[string]any, 0, len(rows))
	for _, d := range rows {
		days = append(days, map[string]any{"dia": d.Day, "tss": d.TSS, "ctl": d.CTL, "atl": d.ATL, "tsb": d.TSB})
	}
	writeJSON(w, http.StatusOK, map[string]any{"atleta_id": id, "dias": days})
}

func (a *API) getActivities(w http.ResponseWriter, r *http.Request, p athlete.Principal) {
	q, err := parseWindow(r)
	if err != nil {
		problem(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	rows, err := a.svc.Activities(r.Context(), p, q.athlete, q.from, q.to)
	if err != nil {
		fail(w, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, x := range rows {
		items = append(items, map[string]any{
			"atividade_id": x.ActivityID, "dia": x.Day, "tipo": x.Type, "metodo": x.Method, "tss": x.TSS,
			"np": x.NP, "fator_intensidade": x.IntensityFactor, "descartes_fc": x.DiscardedHR, "descartes_potencia": x.DiscardedPower,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"itens": items})
}

type thresholdBody struct {
	Kind      string  `json:"tipo"`
	Value     float64 `json:"valor"`
	ValidFrom string  `json:"vigente_desde"`
}

func (a *API) getThresholds(w http.ResponseWriter, r *http.Request, p athlete.Principal) {
	id, err := athleteParam(r)
	if err != nil {
		problem(w, http.StatusUnprocessableEntity, "atleta_id inválido")
		return
	}
	rows, err := a.svc.Thresholds(r.Context(), p, id)
	if err != nil {
		fail(w, err)
		return
	}
	items := make([]thresholdBody, 0, len(rows))
	for _, t := range rows {
		items = append(items, thresholdBody{t.Kind, t.Value, t.ValidFrom})
	}
	writeJSON(w, http.StatusOK, map[string]any{"itens": items})
}

func (a *API) putThreshold(w http.ResponseWriter, r *http.Request, p athlete.Principal) {
	var b thresholdBody
	if !decode(w, r, &b) {
		return
	}
	if err := a.svc.PutThreshold(r.Context(), p, athlete.Threshold{Kind: b.Kind, Value: b.Value, ValidFrom: b.ValidFrom}); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

type profileBody struct {
	Sex       string `json:"sexo"`
	RestingHR int    `json:"fc_repouso"`
	MaxHR     int    `json:"fc_max"`
}

func (a *API) getProfile(w http.ResponseWriter, r *http.Request, p athlete.Principal) {
	id, err := athleteParam(r)
	if err != nil {
		problem(w, http.StatusUnprocessableEntity, "atleta_id inválido")
		return
	}
	pr, err := a.svc.Profile(r.Context(), p, id)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profileBody{pr.Sex, pr.RestingHR, pr.MaxHR})
}

func (a *API) putProfile(w http.ResponseWriter, r *http.Request, p athlete.Principal) {
	var b profileBody
	if !decode(w, r, &b) {
		return
	}
	if err := a.svc.PutProfile(r.Context(), p, athlete.Profile{Sex: b.Sex, RestingHR: b.RestingHR, MaxHR: b.MaxHR}); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		problem(w, http.StatusUnprocessableEntity, "corpo inválido")
		return false
	}
	return true
}

func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, athlete.ErrNotFound):
		problem(w, http.StatusNotFound, "Não encontrado")
	case errors.Is(err, athlete.ErrForbidden):
		problem(w, http.StatusForbidden, "Papel sem permissão")
	case errors.Is(err, athlete.ErrInvalid):
		problem(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, jwtauth.ErrUnauthenticated):
		problem(w, http.StatusUnauthorized, "Não autenticado")
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
