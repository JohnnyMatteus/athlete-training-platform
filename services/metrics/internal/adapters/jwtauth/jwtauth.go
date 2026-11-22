// Package jwtauth é a ACL que valida o JWT RS256 (ADR 0011); nenhum tipo da biblioteca atravessa a porta.
package jwtauth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/athlete"
)

// ErrUnauthenticated cobre token ausente, malformado, expirado ou com iss/aud errados.
var ErrUnauthenticated = errors.New("não autenticado")

// Authenticator valida tokens com a chave pública configurada.
type Authenticator struct {
	key      *rsa.PublicKey
	issuer   string
	audience string
}

// FromFile lê a chave pública PEM.
func FromFile(path, issuer, audience string) (*Authenticator, error) {
	pem, err := os.ReadFile(path) //nolint:gosec // caminho vem da configuração do operador
	if err != nil {
		return nil, fmt.Errorf("ler chave pública: %w", err)
	}
	return New(pem, issuer, audience)
}

func New(pem []byte, issuer, audience string) (*Authenticator, error) {
	key, err := jwt.ParseRSAPublicKeyFromPEM(pem)
	if err != nil {
		return nil, fmt.Errorf("chave pública inválida: %w", err)
	}
	return &Authenticator{key: key, issuer: issuer, audience: audience}, nil
}

type claims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// Authenticate devolve o chamador; roles ausente equivale a ["atleta"].
func (a *Authenticator) Authenticate(token string) (athlete.Principal, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return a.key, nil },
		jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(a.issuer), jwt.WithAudience(a.audience), jwt.WithExpirationRequired())
	if err != nil {
		return athlete.Principal{}, fmt.Errorf("%w: %v", ErrUnauthenticated, err)
	}
	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return athlete.Principal{}, fmt.Errorf("%w: sub não é UUID", ErrUnauthenticated)
	}
	roles := c.Roles
	if len(roles) == 0 {
		roles = []string{athlete.RoleAthlete}
	}
	return athlete.Principal{ID: id, Roles: roles}, nil
}
