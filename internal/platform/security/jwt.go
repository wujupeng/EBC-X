package security

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken      = errors.New("invalid JWT token")
	ErrExpiredToken      = errors.New("JWT token expired")
	ErrMissingToken      = errors.New("missing JWT token")
	ErrInvalidIssuer     = errors.New("invalid JWT issuer")
	ErrInvalidAudience   = errors.New("invalid JWT audience")
	ErrInsufficientScope = errors.New("insufficient scope for operation")
	ErrTokenRevoked      = errors.New("JWT token has been revoked")
)

type Claims struct {
	TenantID string   `json:"tenant_id"`
	UserID   string   `json:"user_id"`
	OrgID    string   `json:"org_id"`
	Roles    []string `json:"roles"`
	Scopes   []string `json:"scopes"`
	jwt.RegisteredClaims
}

type AuthGateway struct {
	signingKey     []byte
	issuer         string
	audience       string
	accessTokenTTL time.Duration
	revokedTokens  map[string]time.Time
}

func NewAuthGateway(signingKey []byte, issuer, audience string, accessTokenTTL time.Duration) *AuthGateway {
	return &AuthGateway{
		signingKey:     signingKey,
		issuer:         issuer,
		audience:       audience,
		accessTokenTTL: accessTokenTTL,
		revokedTokens:  make(map[string]time.Time),
	}
}

func (g *AuthGateway) IssueToken(tenantID, userID, orgID string, roles, scopes []string) (string, error) {
	now := time.Now()
	claims := Claims{
		TenantID: tenantID,
		UserID:   userID,
		OrgID:    orgID,
		Roles:    roles,
		Scopes:   scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    g.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{g.audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(g.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(g.signingKey)
}

func (g *AuthGateway) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrMissingToken
	}

	if _, revoked := g.revokedTokens[tokenString]; revoked {
		return nil, ErrTokenRevoked
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return g.signingKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Issuer != g.issuer {
		return nil, ErrInvalidIssuer
	}

	if len(claims.Audience) == 0 || claims.Audience[0] != g.audience {
		return nil, ErrInvalidAudience
	}

	return claims, nil
}

func (g *AuthGateway) RevokeToken(tokenString string) {
	g.revokedTokens[tokenString] = time.Now()
}

func (g *AuthGateway) CheckScope(claims *Claims, requiredScope string) error {
	for _, s := range claims.Scopes {
		if s == requiredScope || s == "*" {
			return nil
		}
	}
	return ErrInsufficientScope
}

func (g *AuthGateway) CheckRole(claims *Claims, requiredRole string) error {
	for _, r := range claims.Roles {
		if r == requiredRole || r == "admin" {
			return nil
		}
	}
	return ErrInsufficientScope
}

func ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", ErrMissingToken
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", ErrInvalidToken
	}
	return parts[1], nil
}
