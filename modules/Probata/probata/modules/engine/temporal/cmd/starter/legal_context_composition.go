package main

import (
	"errors"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
	"net/http"
)

func legalContextHandler(pool platformpostgres.DB, serviceTokenFile string) (http.Handler, error) {
	store, err := platformpostgres.NewLegalContextStore(pool)
	if err != nil {
		return nil, err
	}
	handler, err := runtimeapi.NewLegalContextHTTPHandler(store, serviceTokenFile)
	if err != nil {
		return nil, err
	}
	investigations, err := platformpostgres.NewInvestigationRequestStore(pool)
	if err != nil {
		return nil, err
	}
	receiver, err := runtimeapi.NewInvestigationHTTPHandler(investigations, serviceTokenFile)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle(runtimeapi.LegalContextRoutePattern, handler.Routes())
	mux.Handle(runtimeapi.InvestigationCreateRoutePattern, receiver.Routes())
	mux.Handle(runtimeapi.InvestigationReadRoutePattern, receiver.Routes())
	return mux, nil
}

func mountLegalContextRoutes(existing, legalContext http.Handler) (http.Handler, error) {
	if existing == nil || legalContext == nil {
		return nil, errors.New("legal context requires existing and context handlers")
	}
	mux := http.NewServeMux()
	mux.Handle(runtimeapi.LegalContextRoutePattern, legalContext)
	mux.Handle(runtimeapi.InvestigationCreateRoutePattern, legalContext)
	mux.Handle(runtimeapi.InvestigationReadRoutePattern, legalContext)
	mux.Handle("/", existing)
	return mux, nil
}
