// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metaserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"k8s.io/kubernetes/pkg/serviceaccount"
)

// Logger is a simple logger interface.
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(err error, msg string, keysAndValues ...interface{})
}

// OpenIDMetadataServerOptions are options that can be applied to [OpenIDMetadataServer].
type OpenIDMetadataServerOptions struct {
	// Host is the host that the server will listen on.
	Host string
	// Port is the port that the server will listen on. Default is 443
	Port int
}

// OpenIDMetadataServer is an HTTP server for metadata of the identity token issuer.
type OpenIDMetadataServer struct {
	configJSON []byte
	keysetJSON []byte
	server     *http.Server
	logger     Logger
}

// NewOpenIDMetadataServer creates a new [OpenIDMetadataServer].
// The hostname is the is the OIDC issuer's hostname, publicKeys are the keys
// that may be used to sign identity tokens.
func NewOpenIDMetadataServer(
	issuerURL string,
	publicKeys []interface{},
	tlsConfig *tls.Config,
	logger Logger,
	opts OpenIDMetadataServerOptions,
) (*OpenIDMetadataServer, error) {
	iss, err := url.Parse(issuerURL)
	if err != nil {
		return nil, err
	}
	if len(iss.Path) > 0 {
		return nil, fmt.Errorf("issuer URL may not include path: %s", issuerURL)
	}

	port := 443
	if opts.Port != 0 {
		port = opts.Port
	}

	jwksURI := issuerURL + "/jwks"
	meta, err := serviceaccount.NewOpenIDMetadata(issuerURL, jwksURI, "", publicKeys)
	if err != nil {
		return nil, err
	}

	if tlsConfig == nil || (len(tlsConfig.Certificates) == 0 && tlsConfig.GetCertificate == nil) {
		return nil, errors.New("openid metadata server: tls is not configured")
	}

	s := &OpenIDMetadataServer{
		configJSON: meta.ConfigJSON,
		keysetJSON: meta.PublicKeysetJSON,
		logger:     logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", s.serveConfiguration)
	mux.HandleFunc("/jwks", s.serveKeys)
	mux.HandleFunc("/healthz", s.serveHealthz)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%v", opts.Host, port),
		Handler:      mux,
		TLSConfig:    tlsConfig,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	s.server = server
	return s, nil
}

// Run starts the [OpenIDMetadataServer]. It returns if stopCh is closed or the server cannot start initially.
func (s *OpenIDMetadataServer) Run(stopCh <-chan struct{}) error {
	errCh := make(chan error)
	go func(errCh chan<- error) {
		s.logger.Info("openid metadata server starts listening", "address", s.server.Addr)
		defer close(errCh)
		if err := s.server.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("openid metadata server failed serving content: %w", err)
		} else {
			s.logger.Info("openid metadata server stopped listening")
		}
	}(errCh)

	select {
	case err := <-errCh:
		return err
	case <-stopCh:
		s.logger.Info("openid metadata server shutting down")
		cancelCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		err := s.server.Shutdown(cancelCtx)
		if err != nil {
			return fmt.Errorf("openid metadata server failed graceful shutdown: %w", err)
		}
		s.logger.Info("openid metadata server shutdown successful")
		return nil
	}
}
