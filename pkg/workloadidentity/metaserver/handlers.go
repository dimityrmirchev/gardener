/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Contents of this file were copied and modified from the kubernetes/kubernetes project
https://github.com/kubernetes/kubernetes/blob/246d363ea4bab2ac99a938d0cee73d72fc44de45/pkg/routes/openidmetadata.go
Modifications Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved.
*/

package metaserver

import (
	"fmt"
	"net/http"
)

const (
	headerContentType  = "Content-Type"
	headerCacheControl = "Cache-Control"
	cacheControl       = "public, max-age=3600"

	mimeJSON = "application/json"
	mimeJWKS = "application/jwk-set+json"

	notAllowedTempl = "%s method is not allowed"
)

func (s *OpenIDMetadataServer) serveConfiguration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, fmt.Sprintf(notAllowedTempl, r.Method), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.Header().Set(headerCacheControl, cacheControl)
	if _, err := w.Write(s.configJSON); err != nil {
		s.logger.Error(err, "failed to write workload identity issuer metadata response")
		return
	}
}

func (s *OpenIDMetadataServer) serveKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, fmt.Sprintf(notAllowedTempl, r.Method), http.StatusMethodNotAllowed)
		return
	}

	// Per RFC7517 : https://tools.ietf.org/html/rfc7517#section-8.5.1
	w.Header().Set(headerContentType, mimeJWKS)
	w.Header().Set(headerCacheControl, cacheControl)
	if _, err := w.Write(s.keysetJSON); err != nil {
		s.logger.Error(err, "failed to write workload identity issuer JWKS response")
		return
	}
}

func (s *OpenIDMetadataServer) serveHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, fmt.Sprintf(notAllowedTempl, r.Method), http.StatusMethodNotAllowed)
		return
	}

	if _, err := w.Write([]byte("ok")); err != nil {
		s.logger.Error(err, "failed to write health check response")
		return
	}
}
