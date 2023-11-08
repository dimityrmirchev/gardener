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

package app

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/pkg/workloadidentity/metaserver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/component-base/version"
	"k8s.io/component-base/version/verflag"
)

// Name is a const for the name of this component.
const Name = "gardener-openid-discovery"

// NewCommand creates a new cobra.Command for running gardener-openid-discovery.
func NewCommand() *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   Name,
		Short: Name,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()

			if err := opts.Validate(); err != nil {
				return err
			}

			log, err := logger.NewZapLogger(opts.LogLevel, opts.LogFormat)
			if err != nil {
				return fmt.Errorf("error instantiating zap logger: %w", err)
			}

			log.Info("Starting "+Name, "version", version.Get())
			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Info(fmt.Sprintf("FLAG: --%s=%s", flag.Name, flag.Value)) //nolint:logcheck
			})

			// don't output usage on further errors raised during execution
			cmd.SilenceUsage = true

			serverCert, err := tls.LoadX509KeyPair(opts.TLSCertFile, opts.TLSKeyFile)
			if err != nil {
				return fmt.Errorf("failed to parse openid identity metadata server certificates: %w", err)
			}

			var pubKeys []interface{}
			for _, file := range opts.KeyFiles {
				keys, err := keyutil.PublicKeysFromFile(file)
				if err != nil {
					return fmt.Errorf("failed to parse openid metadata server key file %s: %w", file, err)
				}
				pubKeys = append(pubKeys, keys...)
			}

			server, err := metaserver.NewOpenIDMetadataServer(
				opts.Issuer,
				pubKeys,
				&tls.Config{
					Certificates: []tls.Certificate{serverCert},
					MinVersion:   tls.VersionTLS13,
				},
				log,
				metaserver.OpenIDMetadataServerOptions{Port: opts.Port},
			)
			if err != nil {
				return fmt.Errorf("failed to create a openid metadata server: %w", err)
			}

			return server.Run(cmd.Context().Done())
		},
	}

	flags := cmd.Flags()
	verflag.AddFlags(flags)
	opts.AddFlags(flags)

	return cmd
}

type Options struct {
	LogLevel  string
	LogFormat string

	Issuer      string
	Port        int
	KeyFiles    []string
	TLSCertFile string
	TLSKeyFile  string
}

// AddFlags adds flags related to cluster identity to the options
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Issuer, "issuer", "", "Identifier of the token issuer.")
	fs.IntVar(&o.Port, "port", 11443, "The port on which to serve the metadata documents.")
	fs.StringArrayVar(&o.KeyFiles, "key-file", []string{}, "File containing PEM-encoded x509 RSA or ECDSA private or public keys, used to verify issued tokens. The specified file can contain multiple keys, and the flag can be specified multiple times with different files.")
	fs.StringVar(&o.TLSCertFile, "tls-cert-file", "", "File containing the x509 Certificate for HTTPS used by the server.")
	fs.StringVar(&o.TLSKeyFile, "tls-key-file", "", "File containing the x509 private key matching --tls-cert-file.")

	fs.StringVar(&o.LogLevel, "log-level", "info", "The level/severity for the logs. Must be one of [info,debug,error]")
	fs.StringVar(&o.LogFormat, "log-format", "json", "The format for the logs. Must be one of [json,text]")
}

func (o *Options) Validate() error {
	allErrors := []error{}
	if len(o.KeyFiles) == 0 {
		allErrors = append(allErrors, fmt.Errorf("--key-file must be set at least once"))
	}

	if len(o.TLSCertFile) == 0 {
		allErrors = append(allErrors, fmt.Errorf("--tls-cert-file must be set"))
	}

	if len(o.TLSKeyFile) == 0 {
		allErrors = append(allErrors, fmt.Errorf("--tls-key-file must be set"))
	}

	if !sets.New(logger.AllLogLevels...).Has(o.LogLevel) {
		allErrors = append(allErrors, fmt.Errorf("invalid --log-level: %s", o.LogLevel))
	}

	if !sets.New(logger.AllLogFormats...).Has(o.LogFormat) {
		allErrors = append(allErrors, fmt.Errorf("invalid --log-format: %s", o.LogFormat))
	}

	return errors.Join(allErrors...)
}
