package bic

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/leonnicolas/iban-gen/iban"
)

// Bank represents a bank.
type Bank struct {
	CountryCode iban.CountryCode
	Bank        string
	BankCode    string
	BIC         string
}

// BankRepo contains Banks and enables queries.
type BankRepo struct {
	bics map[string]Bank
}

// NewBICRepo returns a new BankRepo
func NewBICRepo() *BankRepo {
	return &BankRepo{}
}

// BICs returns all BICs of the BankRepo.
func (re *BankRepo) BICs() []Bank {
	ret := make([]Bank, 0, len(re.bics))
	for _, v := range re.bics {
		ret = append(ret, v)
	}
	return ret
}

// BankCode returns the BankCode of the bank of the given BIC.
func (re *BankRepo) BankCode(bic string) (string, bool) {
	b, ok := re.bics[bic]
	return b.BankCode, ok
}

// PopulateFromFile populates the BankRepo from a file.
func (re *BankRepo) PopulateFromFile(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	return re.Populate(f)
}

// Populate populates the BankRepo from a io.Reader.
func (re *BankRepo) Populate(r io.Reader) (int, error) {
	if re.bics == nil {
		re.bics = make(map[string]Bank)
	}
	s := bufio.NewReader(r)
	c := 0
	for l, err := s.ReadString('\n'); err == nil; l, err = s.ReadString('\n') {
		if len(l) < 168 {
			return 0, errors.New("invalid entry")
		}
		runeVal := []rune(l)
		bc := strings.TrimSpace(string(runeVal[0:8]))
		bic := strings.TrimSpace(string(runeVal[139:150]))
		name := strings.TrimSpace(string(runeVal[9:67]))
		re.bics[strings.Trim(bic, " ")] = Bank{
			CountryCode: iban.CountryCodeDE,
			BIC:         bic,
			Bank:        name,
			BankCode:    bc,
		}
		c++
	}
	return c, nil
}
