// Copyright 2020 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
package identity

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// Token field names
const (
	subjectClaim = "sub"
	tenantClaim  = "mender.tenant"
	deviceClaim  = "mender.device"
	userClaim    = "mender.user"
	planClaim    = "mender.plan"
)

type Identity struct {
	Subject  string
	Tenant   string
	IsUser   bool
	IsDevice bool
	Plan     string
}

type rawClaims map[string]interface{}

func decodeClaims(token string) (rawClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("incorrect token format")
	}

	b64claims := parts[1]
	// add padding as needed
	if pad := len(b64claims) % 4; pad != 0 {
		b64claims += strings.Repeat("=", 4-pad)
	}

	rawclaims, err := base64.URLEncoding.DecodeString(b64claims)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode raw claims %v",
			b64claims)
	}

	var claims rawClaims
	err = json.Unmarshal(rawclaims, &claims)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode claims")
	}

	return claims, nil
}

// Generate identity information from given JWT by extracting subject and tenant claims.
// Note that this function does not perform any form of token signature
// verification.
func ExtractIdentity(token string) (Identity, error) {
	claims, err := decodeClaims(token)
	if err != nil {
		return Identity{}, err
	}

	sub, err := getStringClaim(claims, subjectClaim)
	if err != nil {
		return Identity{}, err
	}
	if sub == "" {
		return Identity{}, errors.Errorf("subject claim not found")
	}

	tenant, err := getStringClaim(claims, tenantClaim)
	if err != nil {
		return Identity{}, err
	}

	plan, err := getStringClaim(claims, planClaim)
	if err != nil {
		return Identity{}, err
	}

	identity := Identity{Subject: sub, Tenant: tenant, Plan: plan}
	if isUser, err := getBoolClaim(claims, userClaim); err == nil {
		identity.IsUser = isUser
	}

	if isDevice, err := getBoolClaim(claims, deviceClaim); err == nil {
		identity.IsDevice = isDevice
	}

	return identity, nil
}

// Extract identity information from HTTP Authorization header. The header is
// assumed to contain data in format: `Bearer <token>`
func ExtractIdentityFromHeaders(headers http.Header) (Identity, error) {
	auth := strings.Split(headers.Get("Authorization"), " ")

	if len(auth) != 2 {
		return Identity{}, errors.Errorf("malformed authorization data")
	}

	if auth[0] != "Bearer" {
		return Identity{}, errors.Errorf("unknown authorization method %v", auth[0])
	}

	return ExtractIdentity(auth[1])
}

// extracts claim from JWT claims and converts it to a string
func getStringClaim(claims rawClaims, claim string) (string, error) {
	raw, ok := claims[claim]
	if !ok {
		return "", nil
	}

	claimString, ok := raw.(string)
	if !ok {
		return "", errors.Errorf("invalid %s format", claim)
	}
	return claimString, nil
}

// extracts claim from JWT claims and converts it to a boolean
func getBoolClaim(claims rawClaims, name string) (bool, error) {
	rawval, ok := claims[name]
	if !ok {
		return false, errors.Errorf("field %s not found", name)
	}

	val, ok := rawval.(bool)
	if !ok {
		return false, errors.Errorf("field %v has incorrect value %v", name, rawval)
	}

	return val, nil
}
