package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type JWTVerifier struct {
	publicKey     *rsa.PublicKey
	authServerURL string
}

type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"sub"`
}

func NewJWTVerifier(authServerURL string) (*JWTVerifier, error) {
	verifier := &JWTVerifier{
		authServerURL: authServerURL,
	}

	if err := verifier.fetchPublicKey(); err != nil {
		return nil, fmt.Errorf("failed to fetch public key: %w", err)
	}

	return verifier, nil
}

func (v *JWTVerifier) fetchPublicKey() error {
	url := v.authServerURL + "/auth/public-key"
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch public key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var keyResp struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal(body, &keyResp); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}

	block, _ := pem.Decode([]byte(keyResp.PublicKey))
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("not an RSA public key")
	}

	v.publicKey = rsaPub
	return nil
}

func (v *JWTVerifier) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
