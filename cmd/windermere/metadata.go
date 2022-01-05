/*
 *  This file is part of Windermere (EGIL SCIM Server).
 *
 *  Copyright (C) 2019-2021 Föreningen Sambruk
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU Affero General Public License as
 *  published by the Free Software Foundation, either version 3 of the
 *  License, or (at your option) any later version.

 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU Affero General Public License for more details.

 *  You should have received a copy of the GNU Affero General Public License
 *  along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/joesiltberg/bowness/fedtls"
	bownessutil "github.com/joesiltberg/bowness/util"
)

func publicKeyPin(certPath string) (string, error) {
	pemdata, err := ioutil.ReadFile(certPath)

	if err != nil {
		return "", err
	}

	derdata, _ := pem.Decode(pemdata)

	if derdata == nil {
		return "", errors.New("Failed to decode PEM data when pinning public key")
	}

	x509Cert, err := x509.ParseCertificate(derdata.Bytes)
	if err != nil {
		return "", err
	}

	return bownessutil.Fingerprint(x509Cert), nil
}

// Creates a http.Handler for returning this server's metadata for the TLS federation
func metadataHandler(certPath, entityID, baseURI, organization, organizationID string) http.Handler {
	cert, err := os.ReadFile(certPath)

	if err != nil {
		log.Fatalf("Failed to read certificate when creating metadata: %v", err)
	}

	issuer := fedtls.Issuer{
		X509certificate: string(cert),
	}

	pkp, err := publicKeyPin(certPath)

	if err != nil {
		log.Fatalf("Failed to generate public key pin when creating metadata: %v", err)
	}

	pin := fedtls.Pin{
		Alg:    "sha256",
		Digest: pkp,
	}

	description := "EGIL SCIM server"
	server := fedtls.Server{
		Description: &description,
		BaseURI:     baseURI,
		Tags:        []string{"egilv1"},
		Pins:        []fedtls.Pin{pin},
	}

	entity := fedtls.Entity{
		Issuers:        []fedtls.Issuer{issuer},
		Servers:        []fedtls.Server{server},
		EntityID:       entityID,
		Organization:   &organization,
		OrganizationID: &organizationID,
	}

	md := fedtls.Metadata{
		Version:  "1.0.0",
		Entities: []fedtls.Entity{entity},
	}

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			body, err := json.MarshalIndent(&md, "", "  ")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(body)
		})
}
