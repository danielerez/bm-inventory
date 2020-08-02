package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/security"
	"github.com/jinzhu/gorm"

	"github.com/dgrijalva/jwt-go"
	"github.com/filanov/bm-inventory/pkg/ocm"
	"github.com/sirupsen/logrus"
)

// AuthPayload defines the structure of the JWT payload we expect from
// RHD JWT tokens
type AuthPayload struct {
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Issuer    string `json:"iss"`
	ClientID  string `json:"clientId"`
	IsAdmin   bool   `json:"is_admin"`
}

type Config struct {
	JwkCertURL string `envconfig:"JWKS_URL" default:""`
	JwkCertCA  string `envconfig:"JWKS_CA" default:""`
}

type AuthHandler struct {
	CertURL string
	keyMap  map[string]*rsa.PublicKey
	client  *ocm.Client
	db      *gorm.DB
}

var authHandler = &AuthHandler{}

func InitAuthHandler(certURL string, ocmCLient *ocm.Client, db *gorm.DB) {
	authHandler.client = ocmCLient
	authHandler.CertURL = certURL
	authHandler.db = db
	err := authHandler.populateKeyMap()
	if err != nil {
		logrus.Fatalln("Failed to init auth manager,", err)
	}
}

func (a *AuthHandler) populateKeyMap() error {
	// Load the trusted CA certificates:
	trustedCAs, err := x509.SystemCertPool()
	if err != nil {
		return fmt.Errorf("can't load system trusted CAs: %v", err)
	}

	// Try to read the JWT public key object file.
	a.keyMap, err = downloadPublicKeys(a.CertURL, trustedCAs)
	return err
}

func (a *AuthHandler) getValidationToken(token *jwt.Token) (interface{}, error) {
	// Try to get the token kid.
	kid, ok := token.Header["kid"]
	if !ok {
		return nil, fmt.Errorf("no kid found in jwt token")
	}

	// Try to get currect cert from certs map.
	key, ok := a.keyMap[kid.(string)]
	if !ok {
		return nil, fmt.Errorf("No matching key in auth keymap for key id [%v]", kid)
	}

	return key, nil
}

func AuthAgentAuth(token string) (interface{}, error) {
	if authHandler.client == nil {
		return nil, fmt.Errorf("OCM client unavailable")
	}
	//TODO: Validate agent pull secret
	logrus.Error("AuthAgentAuth")
	authHandler.client.Authentication.AuthenticatePullSecret(context.Background(), "")
	return "", nil
}

func parsePayload(userToken *jwt.Token) (*AuthPayload, error) {
	claims, ok := userToken.Claims.(jwt.MapClaims)
	if !ok {
		err := fmt.Errorf("Unable to parse JWT token claims")
		return nil, err
	}

	payload := &AuthPayload{}
	// default to the values we expect from RHSSO
	payload.Username, _ = claims["username"].(string)
	payload.FirstName, _ = claims["first_name"].(string)
	payload.LastName, _ = claims["last_name"].(string)
	payload.Email, _ = claims["email"].(string)
	payload.ClientID, _ = claims["clientId"].(string)

	// Check values, if empty, use alternative claims from RHD
	if payload.Username == "" {
		payload.Username, _ = claims["preferred_username"].(string)
	}

	if payload.FirstName == "" {
		payload.FirstName, _ = claims["given_name"].(string)
	}

	if payload.LastName == "" {
		payload.LastName, _ = claims["family_name"].(string)
	}

	// If given and family names are not present, use the name field
	if payload.FirstName == "" || payload.LastName == "" {
		name, _ := claims["name"].(string)
		names := strings.Split(name, " ")
		if len(names) > 1 {
			payload.FirstName = names[0]
			payload.LastName = names[1]
		} else {
			payload.FirstName = names[0]
		}
	}
	// TODO: cache admin capability in memory
	admin, err := IsAdmin(payload.Username)
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch user's capabilities: %v", err)
	}
	payload.IsAdmin = admin

	return payload, nil
}

func AuthUserAuth(token string) (interface{}, error) {
	// Handle Bearer
	authHeaderParts := strings.Fields(token)
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return nil, fmt.Errorf("Authorization header format must be Bearer {token}")
	}
	// Now parse the token
	parsedToken, err := jwt.Parse(authHeaderParts[1], authHandler.getValidationToken)

	// Check if there was an error in parsing...
	if err != nil {
		logrus.Errorf("Error parsing token: %v", err)
		return nil, fmt.Errorf("Error parsing token: %v", err)
	}

	if jwt.SigningMethodRS256 != nil && jwt.SigningMethodRS256.Alg() != parsedToken.Header["alg"] {
		message := fmt.Sprintf("Expected %s signing method but token specified %s",
			jwt.SigningMethodRS256.Alg(),
			parsedToken.Header["alg"])
		logrus.Errorf("Error validating token algorithm: %s", message)
		return nil, fmt.Errorf("Error validating token algorithm: %s", message)
	}

	// Check if the parsed token is valid...
	if !parsedToken.Valid {
		logrus.Error("Token is invalid")
		return nil, fmt.Errorf("Token is invalid: %v", parsedToken)
	}

	payload, err := parsePayload(parsedToken)
	if err != nil {
		logrus.Error("Failed parse payload,", err)
		return nil, err
	}

	return payload, nil
}

func CreateAuthenticator(isEnabled bool) func(name, in string, authenticate security.TokenAuthentication) runtime.Authenticator {
	return func(name string, _ string, authenticate security.TokenAuthentication) runtime.Authenticator {
		getToken := func(r *http.Request) string { return r.Header.Get(name) }

		return security.HttpAuthenticator(func(r *http.Request) (bool, interface{}, error) {

			if !isEnabled {
				return true, "", nil
			}
			token := getToken(r)
			if token == "" {
				return false, nil, nil
			}
			p, err := authenticate(token)
			return true, p, err
		})
	}
}