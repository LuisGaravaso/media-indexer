package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// ContextKeyUserID is the Gin context key for authenticated user ID (UUID string).
	ContextKeyUserID = "userID"
	// ContextKeyUserEmail is the Gin context key for authenticated user email.
	ContextKeyUserEmail = "userEmail"
	// ContextKeyUserRole is the Gin context key for authenticated user role.
	ContextKeyUserRole = "userRole"
)

// SupabaseClaims represents the standard and custom claims present in Supabase Auth JWTs.
type SupabaseClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email,omitempty"`
	Role  string `json:"role,omitempty"`
}

// JWK represents a JSON Web Key in a JWKS response.
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWKSValidator validates JWTs against a remote JWKS endpoint or HMAC secret with caching.
type JWKSValidator struct {
	jwksURL    string
	jwtSecret  string
	httpClient *http.Client
	mu         sync.RWMutex
	keys       map[string]interface{}
	lastFetch  time.Time
}

// NewJWKSValidator creates a new JWKS validator instance.
func NewJWKSValidator(supabaseURL string, jwtSecret string) *JWKSValidator {
	var jwksURL string
	if supabaseURL != "" {
		jwksURL = strings.TrimRight(supabaseURL, "/") + "/auth/v1/.well-known/jwks.json"
	}
	return &JWKSValidator{
		jwksURL:    jwksURL,
		jwtSecret:  jwtSecret,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		keys:       make(map[string]interface{}),
	}
}

// KeyFunc dynamically resolves the verification key for the JWT.
func (v *JWKSValidator) KeyFunc(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
		if v.jwtSecret != "" {
			return []byte(v.jwtSecret), nil
		}
		return nil, errors.New("HMAC signing secret not configured")
	}

	kid, _ := token.Header["kid"].(string)

	v.mu.RLock()
	key, exists := v.keys[kid]
	needsRefresh := time.Since(v.lastFetch) > 10*time.Minute || (!exists && kid != "")
	v.mu.RUnlock()

	if !exists || needsRefresh {
		if err := v.refreshKeys(); err != nil && !exists {
			return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
		}
		v.mu.RLock()
		key, exists = v.keys[kid]
		v.mu.RUnlock()
	}

	if exists {
		return key, nil
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	for _, k := range v.keys {
		return k, nil
	}

	return nil, errors.New("no matching key found for token verification")
}

func (v *JWKSValidator) refreshKeys() error {
	if v.jwksURL == "" {
		return errors.New("JWKS URL is not configured")
	}

	req, err := http.NewRequest(http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	newKeys := make(map[string]interface{})
	for _, jwk := range jwks.Keys {
		if jwk.Kty == "EC" && jwk.Crv == "P-256" {
			xBytes, errX := base64.RawURLEncoding.DecodeString(jwk.X)
			yBytes, errY := base64.RawURLEncoding.DecodeString(jwk.Y)
			if errX == nil && errY == nil {
				pubKey := &ecdsa.PublicKey{
					Curve: elliptic.P256(),
					X:     new(big.Int).SetBytes(xBytes),
					Y:     new(big.Int).SetBytes(yBytes),
				}
				newKeys[jwk.Kid] = pubKey
			}
		}
	}

	v.mu.Lock()
	v.keys = newKeys
	v.lastFetch = time.Now()
	v.mu.Unlock()

	return nil
}

// RequireAuth returns a Gin middleware that validates Supabase JWT Bearer tokens.
func RequireAuth(validator *JWKSValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if validator == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "server auth misconfiguration",
			})
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "malformed authorization header, expected 'Bearer <token>'",
			})
			return
		}

		tokenString := strings.TrimSpace(parts[1])

		var claims SupabaseClaims
		token, err := jwt.ParseWithClaims(tokenString, &claims, validator.KeyFunc)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		userID := claims.Subject
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token subject (user ID) missing",
			})
			return
		}

		c.Set(ContextKeyUserID, userID)
		c.Set("user_id", userID)
		if claims.Email != "" {
			c.Set(ContextKeyUserEmail, claims.Email)
		}
		if claims.Role != "" {
			c.Set(ContextKeyUserRole, claims.Role)
		}

		c.Next()
	}
}

// GetUserID retrieves the authenticated user's ID from the Gin context.
func GetUserID(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextKeyUserID)
	if !exists {
		return "", false
	}
	userID, ok := val.(string)
	return userID, ok
}
