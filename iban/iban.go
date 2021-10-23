package iban

import (
	"fmt"
	"math/big"
	"math/rand"
	"regexp"
	"time"
)

type CountryCode string

const (
	CountryCodeDE = "DE"
)

var random = rand.New(rand.NewSource(time.Now().Unix()))

type IBAN struct {
	bc  string
	bic string
	aNo string
	cc  CountryCode
	cs  string
}

func GenerateForCountry(cc CountryCode) (*IBAN, error) {
	return GenerateFromBankCode(cc, randomNoString(8))
}

func GenerateFromBankCode(cc CountryCode, bc string) (*IBAN, error) {
	if cc == CountryCodeDE && len(bc) != 8 {
		return nil, fmt.Errorf("bank code must be %d charackters for %s", 8, string(cc))
	}
	return IBAN{
		bc:  bc,
		aNo: randomNoString(10),
		cc:  cc,
	}.check()
}

func (i *IBAN) BIC() string {
	return i.bic
}

func (i *IBAN) BankCode() string {
	return i.bc
}

func (i *IBAN) AccountNo() string {
	return i.aNo
}

func (i *IBAN) CountryCode() string {
	return string(i.cc)
}

func (i *IBAN) String() string {
	return fmt.Sprintf("%s%s%s%s", i.cc, i.cs, i.bc, i.aNo)
}

func randomNoString(l uint) (s string) {
	for i := uint(0); i < l; i++ {
		s = fmt.Sprintf("%s%d", s, random.Intn(10))
	}
	return
}

func (i IBAN) check() (*IBAN, error) {
	b, ok := big.NewInt(0).SetString(fmt.Sprintf("%s%s%s00", i.bc, i.aNo, i.cc.ToNum()), 10)
	if !ok {
		return nil, fmt.Errorf("failed to convert bank account number %q to big int.", i.bc)
	}
	i.cs = fmt.Sprintf("%02d", big.NewInt(0).Sub(big.NewInt(98), b.Mod(b, big.NewInt(97))).Int64())
	return &i, nil
}

var countryCode = regexp.MustCompile(`^[A-Z]{2}$`)

func (c CountryCode) ToNum() string {
	return "1314"
}
