package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Sub      string `json:"sub"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type JWTVerifier struct {
	publicKey     *rsa.PublicKey
	authServerURL string
}

type PublicKeyResponse struct {
	PublicKey string `json:"public_key"`
	Algorithm string `json:"algorithm"`
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

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch public key: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var keyResp PublicKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&keyResp); err != nil {
		return err
	}

	block, _ := pem.Decode([]byte(keyResp.PublicKey))
	if block == nil {
		return errors.New("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return errors.New("not an RSA public key")
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

	return nil, errors.New("invalid token")
}

func (v *JWTVerifier) RefreshPublicKey() error {
	return v.fetchPublicKey()
}
