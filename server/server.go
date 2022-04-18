package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/agnivade/levenshtein"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/metalmatze/signal/server/signalhttp"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slices"

	v1 "github.com/leonnicolas/iban-gen/api/v1"
	"github.com/leonnicolas/iban-gen/bic"
	"github.com/leonnicolas/iban-gen/iban"
)

type instrumentedServer struct {
	server
	instrumenter signalhttp.HandlerInstrumenter
}

// NewInstrumentedServerWithLogger returns a Server that has been instrumented with prometheus.
func NewInstrumentedServerWithLogger(bicsRepo *bic.BankRepo, r prometheus.Registerer, logger log.Logger) v1.ServerInterface {
	return &instrumentedServer{
		instrumenter: signalhttp.NewHandlerInstrumenter(r, []string{"handler"}),
		server:       newWithLogger(bicsRepo, logger),
	}
}

// Random returns a random iban.
func (s *instrumentedServer) Random(w http.ResponseWriter, r *http.Request, params v1.RandomParams) {
	h := s.instrumenter.NewHandler(
		prometheus.Labels{"handler": "random"},
		http.HandlerFunc(s.server.random(w, r, params)),
	)
	h(w, r)
}

// Bics returns BICs.
func (s *instrumentedServer) Bics(w http.ResponseWriter, r *http.Request, params v1.BicsParams) {
	h := s.instrumenter.NewHandler(
		prometheus.Labels{"handler": "bics"},
		http.HandlerFunc(s.server.bics(w, r, params)),
	)
	h(w, r)
}

// CountryCodes returns all CountryCodes.
func (s *instrumentedServer) CountryCodes(w http.ResponseWriter, r *http.Request) {
	s.instrumenter.NewHandler(
		prometheus.Labels{"handler": "bics"},
		http.HandlerFunc(s.server.countryCodes(w, r)),
	)(w, r)
}

type server struct {
	bicsRepo  *bic.BankRepo
	logger    log.Logger
	httpError func(w http.ResponseWriter, m string, code int)
}

// newWithLogger returns a new Server.
func newWithLogger(bicsRepo *bic.BankRepo, logger log.Logger) server {
	return server{bicsRepo, logger, httpError(logger)}
}

func httpError(logger log.Logger) func(w http.ResponseWriter, m string, code int) {
	return func(w http.ResponseWriter, m string, code int) {
		res := v1.Error{
			Error: m,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			level.Error(logger).Log("msg", "failed to write response", "err", err.Error())
		}
	}
}

// random returns a random iban.
func (s *server) random(w http.ResponseWriter, r *http.Request, params v1.RandomParams) func(rw http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var i *iban.IBAN
		var err error
		if params.Bic != nil && *params.Bic != "" {
			bc, ok := s.bicsRepo.BankCode(*params.Bic)
			if !ok {
				s.httpError(w, "unknown bic", http.StatusNotFound)
				return
			}
			i, err = iban.GenerateFromBankCode(iban.CountryCodeDE, bc)
			if err != nil {
				s.httpError(w, err.Error(), http.StatusInternalServerError)
				return

			}
		} else if params.BankCode != nil && *params.BankCode != "" {
			i, err = iban.GenerateFromBankCode(iban.CountryCodeDE, *params.BankCode)
			if err != nil {
				s.httpError(w, err.Error(), http.StatusBadRequest)
				return

			}
		} else {
			i, err = iban.GenerateForCountry(iban.CountryCodeDE)
			if err != nil {
				s.httpError(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		res := v1.IBANGeneration{
			Bankcode: i.BankCode(),
			Iban:     i.String(),
			Bic:      params.Bic,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(res); err != nil {
			s.httpError(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// bics returns BICs.
func (s *server) bics(w http.ResponseWriter, r *http.Request, params v1.BicsParams) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		bics := s.bicsRepo.BICs()
		res := make([]v1.BIC, len(bics))
		for i, v := range bics {
			res[i] = v1.BIC{
				CountryCode: string(v.CountryCode),
				Bic:         v.BIC,
				Bank:        v.Bank,
			}
		}
		if params.Bank != nil && *params.Bank != "" {
			res = filter(res, func(b v1.BIC) bool {
				return strings.Contains(strings.ToLower(b.Bank), strings.ToLower(*params.Bank))
			})
			slices.SortFunc(res, func(a, b v1.BIC) bool {
				return levenshtein.ComputeDistance(strings.ToLower(a.Bank), strings.ToLower(*params.Bank)) < levenshtein.ComputeDistance(strings.ToLower(b.Bank), strings.ToLower(*params.Bank))
			})
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(res); err != nil {
			s.httpError(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// countryCodes returns all countryCodes.
func (s *server) countryCodes(w http.ResponseWriter, r *http.Request) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		res := make([]string, 1)
		res[0] = string(iban.CountryCodeDE)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(res); err != nil {
			s.httpError(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func filter[T any](s []T, f func(a T) bool) []T {
	ret := make([]T, 0, len(s))
	i := 0
	for _, e := range s {
		if f(e) {
			ret = append(ret, e)
			i++
		}
	}
	return ret
}
