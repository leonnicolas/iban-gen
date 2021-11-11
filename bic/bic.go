package bic

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/leonnicolas/iban-gen/iban"
)

type BIC struct {
	CountryCode iban.CountryCode
	Bank        string
	BankCode    string
	BIC         string
}

type BICRepo struct {
	bics map[string]BIC
}

func NewBICRepo() *BICRepo {
	return &BICRepo{}
}

func (re *BICRepo) BICs() []BIC {
	ret := make([]BIC, 0, len(re.bics))
	for _, v := range re.bics {
		ret = append(ret, v)
	}
	return ret
}

func (re *BICRepo) BankCode(bic string) (string, bool) {
	b, ok := re.bics[bic]
	return b.BankCode, ok
}

func (re *BICRepo) PopulateFromFile(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	return re.Populate(f)
}

func (re *BICRepo) Populate(r io.Reader) (int, error) {
	if re.bics == nil {
		re.bics = make(map[string]BIC)
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
		re.bics[strings.Trim(bic, " ")] = BIC{
			CountryCode: iban.CountryCodeDE,
			BIC:         bic,
			Bank:        name,
			BankCode:    bc,
		}
		c++
	}
	return c, nil
}
