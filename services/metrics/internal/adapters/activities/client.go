// Package activities lê as amostras de uma atividade no endpoint interno do contexto de Atividades (ADR 0015).
package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/metrics"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

// ErrNotFound indica atividade inexistente no contexto de Atividades.
var ErrNotFound = errors.New("atividade não encontrada")

// Client implementa metrics.SampleSource sobre HTTP, paginando por cursor.
type Client struct {
	base  string
	token string
	page  int
	http  *http.Client
}

var _ metrics.SampleSource = (*Client)(nil)

func NewClient(baseURL, token string) *Client {
	return &Client{
		base: strings.TrimRight(baseURL, "/"), token: token, page: 5000,
		http: &http.Client{Timeout: 30 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
}

type wirePage struct {
	Samples []struct {
		TS       time.Time `json:"ts"`
		HR       *int      `json:"fc_bpm"`
		Power    *int      `json:"potencia_w"`
		Speed    *float64  `json:"velocidade"`
		Altitude *float64  `json:"altitude_m"`
	} `json:"amostras"`
	Next *string `json:"proximo_cursor"`
}

func (c *Client) Samples(ctx context.Context, activityID uuid.UUID) ([]load.Sample, error) {
	var out []load.Sample
	cursor := ""
	for {
		p, err := c.fetch(ctx, activityID, cursor)
		if err != nil {
			return nil, err
		}
		for _, s := range p.Samples {
			out = append(out, load.Sample{At: s.TS, HR: s.HR, Power: s.Power, Speed: s.Speed, Altitude: s.Altitude})
		}
		if p.Next == nil || *p.Next == "" {
			return out, nil
		}
		cursor = *p.Next
	}
}

func (c *Client) fetch(ctx context.Context, id uuid.UUID, cursor string) (wirePage, error) {
	q := url.Values{"limite": {fmt.Sprint(c.page)}}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/internal/atividades/%s/amostras?%s", c.base, id, q.Encode()), nil)
	if err != nil {
		return wirePage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return wirePage{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return wirePage{}, ErrNotFound
	default:
		return wirePage{}, fmt.Errorf("atividades respondeu %d", resp.StatusCode)
	}
	var p wirePage
	return p, json.NewDecoder(resp.Body).Decode(&p)
}
