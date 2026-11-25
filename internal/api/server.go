// SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	goahttp "goa.design/goa/v3/http"

	"github.com/jraddaoui/provenant-passage/internal/api/gen/aips"
	"github.com/jraddaoui/provenant-passage/internal/api/gen/http/aips/server"
)

func Handler(svc aips.Service, logger logr.Logger) http.Handler {
	mux := goahttp.NewMuxer()
	h := server.New(aips.NewEndpoints(svc), mux, goahttp.RequestDecoder, goahttp.ResponseEncoder,
		func(ctx context.Context, w http.ResponseWriter, err error) { logger.Error(err, "HTTP transport error") }, nil)
	server.Mount(mux, h)
	return mux
}

func Server(address string, svc aips.Service, logger logr.Logger) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           Handler(svc, logger),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}
