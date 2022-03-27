# iban-gen

## Hosted Version

The backend is hosted on [iban-gen.duckdns.org](https://iban-gen.duckdns.org) right now.

A frontend is hosted on [iban-gen.klump.solutions](https://iban-gen.klump.solutions).

## API

```shell
curl https://iban-gen.duckdns.org/v1/random
```
or
```shell
curl https://iban-gen.duckdns.org/v1/random?bankCode=01009000
```
or
```shell
curl https://iban-gen.duckdns.org/v1/random?bic=BEVODEBBXXX
```
Check all available BICs with
```shell
curl https://iban-gen.duckdns.org/v1/bics
```
