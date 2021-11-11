package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/metalmatze/signal/server/signalhttp"
	"github.com/prometheus/client_golang/prometheus"

	v1 "github.com/leonnicolas/iban-gen/api/v1"
	"github.com/leonnicolas/iban-gen/bic"
	"github.com/leonnicolas/iban-gen/iban"
)

type instrumentedServer struct {
	server
	instrumenter signalhttp.HandlerInstrumenter
}

func NewInstrumentedServerWithLogger(bicsRepo *bic.BICRepo, r prometheus.Registerer, logger log.Logger) *instrumentedServer {
	return &instrumentedServer{
		instrumenter: signalhttp.NewHandlerInstrumenter(r, []string{"handler"}),
		server:       NewWithLogger(bicsRepo, logger),
	}
}
func (s *instrumentedServer) Random(w http.ResponseWriter, r *http.Request, params v1.RandomParams) {
	h := s.instrumenter.NewHandler(
		prometheus.Labels{"handler": "random"},
		http.HandlerFunc(s.server.Random(w, r, params)),
	)
	h(w, r)
}

func (s *instrumentedServer) Bics(w http.ResponseWriter, r *http.Request, params v1.BicsParams) {
	h := s.instrumenter.NewHandler(
		prometheus.Labels{"handler": "bics"},
		http.HandlerFunc(s.server.Bics(w, r, params)),
	)
	h(w, r)
}

func (s *instrumentedServer) CountryCodes(w http.ResponseWriter, r *http.Request) {
	s.instrumenter.NewHandler(
		prometheus.Labels{"handler": "bics"},
		http.HandlerFunc(s.server.CountryCodes(w, r)),
	)(w, r)
}

type server struct {
	bicsRepo  *bic.BICRepo
	logger    log.Logger
	httpError func(w http.ResponseWriter, m string, code int)
}

func New(bicsRepo *bic.BICRepo) server {
	return NewWithLogger(bicsRepo, log.NewNopLogger())
}

func NewWithLogger(bicsRepo *bic.BICRepo, logger log.Logger) server {
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

func (s *server) Random(w http.ResponseWriter, r *http.Request, params v1.RandomParams) func(rw http.ResponseWriter, r *http.Request) {
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

func (s *server) Bics(w http.ResponseWriter, r *http.Request, params v1.BicsParams) func(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(res); err != nil {
			s.httpError(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (s *server) CountryCodes(w http.ResponseWriter, r *http.Request) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		res := make([]string, 1)
		res[0] = string(iban.CountryCodeDE)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(res); err != nil {
			s.httpError(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
