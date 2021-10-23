package iban

import (
	"testing"
)

func TestCheck(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   IBAN
		out  IBAN
	}{
		{
			name: "01",
			in: IBAN{
				bc:  "10090000",
				aNo: "0000000001",
				cc:  CountryCodeDE,
			},
			out: IBAN{
				bc:  "10090000",
				aNo: "0000000001",
				cc:  CountryCodeDE,
				cs:  "72",
			},
		},
	} {
		out, err := tc.in.check()
		if err != nil {
			t.Errorf("%s got err=%q\n", tc.name, err.Error())
		}
		if *out != tc.out {
			t.Errorf("%s: got=%v expected=%v\n", tc.name, *out, tc.out)
		}
	}
}

func TestNewIBAN(t *testing.T) {
	i, err := GenerateForCountry(CountryCodeDE)
	if err != nil {
		t.Errorf("got err=%q\n", err.Error())
	}
	if i2, _ := i.check(); i.cs != i2.cs {
		t.Errorf("got=%q expected=%q\n", i.cs, i2.cs)
	}
}
