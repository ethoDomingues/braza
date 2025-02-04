package braza

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Session struct {
	claims           jwt.MapClaims
	del              []string
	changed          bool
	expires          time.Time
	Permanent        bool
	expiresPermanent time.Time
}

// validate a cookie session
func (s *Session) validate(c *http.Cookie, ctx *Ctx) {
	secret := ctx.App.SecretKey
	pubKey := ctx.App.SessionPublicKey
	privKey := ctx.App.SessionPrivateKey

	s.claims = jwt.MapClaims{}
	if secret == "" && pubKey == nil && privKey == nil {
		return
	}
	tkn, err := jwt.Parse(c.Value, func(t *jwt.Token) (interface{}, error) { return pubKey, nil })
	if err != nil {
		return
	}

	if claims, ok := tkn.Claims.(jwt.MapClaims); ok && tkn.Valid {
		s.claims = claims
		if p, ok := claims["_permanent"]; ok && p == "true" {
			s.Permanent = true
		}
	}
}

// This inserts a value into the session
func (s *Session) Set(key, value string) {
	if key == "_permanent" {
		panic("'_permanent' is a internal key")
	}
	s.claims[key] = value
	s.changed = true
}

// Returns a session value based on the key. If key does not exist, returns an empty string
func (s *Session) Get(key string) string {
	if v, ok := s.claims[key]; ok {
		return v.(string)
	}
	return ""

}

// Delete a Value from Session
func (s *Session) Del(key string) string {
	if key == "_permanent" {
		panic("'_permanent' is a internal key")
	}
	if v, ok := s.claims[key]; ok {
		s.del = append(s.del, key)
		delete(s.claims, key)
		s.changed = true
		return v.(string)
	}
	return ""
}

// Returns a cookie, with the value being a jwt
func (s *Session) save(ctx *Ctx) *http.Cookie {
	secret := ctx.App.SecretKey
	pubKey := ctx.App.SessionPublicKey
	privKey := ctx.App.SessionPrivateKey
	if secret == "" && pubKey == nil && privKey == nil {
		l.warn.Println("to use the session you need to set a 'App.Secret' or a 'public/private key'. rejecting session")
		return nil
	}
	delete(s.claims, "exp")
	delete(s.claims, "iat")
	delete(s.claims, "_permanent")

	if len(s.claims) == 0 {
		return &http.Cookie{
			Name:     "_session",
			Value:    "",
			HttpOnly: true,
			MaxAge:   -1,
		}
	}
	var exp time.Time
	if s.Permanent {
		if s.expiresPermanent.IsZero() {
			exp = time.Now().Add(time.Hour * 24 * 31)
		} else {
			exp = s.expiresPermanent
		}
		s.claims["_permanent"] = true
	} else {
		if s.expires.IsZero() {
			exp = time.Now().Add(time.Hour)
		} else {
			exp = s.expires
		}
	}
	tkn, err := s.GetSign(ctx)
	if err != nil {
		l.err.Println(err)
		return nil
	}
	c := &http.Cookie{
		Name:     "_session",
		Value:    tkn,
		HttpOnly: true,
		Expires:  exp,
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
	}
	return c
}

// Returns a JWT Token from session data
func (s *Session) GetSign(ctx *Ctx) (string, error) {
	secret := ctx.App.SecretKey
	pubKey := ctx.App.SessionPublicKey
	privKey := ctx.App.SessionPrivateKey
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, s.claims)
	if secret == "" && pubKey == nil && privKey == nil {
		return "", errors.New("to set a session value, you need a set a 'App.Secret' or a 'public/private key'")
	}
	if pubKey != nil && privKey != nil {
		return token.SignedString(privKey)
	}
	return token.SignedString([]byte(secret))
}
