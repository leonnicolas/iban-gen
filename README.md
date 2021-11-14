# iban-gen

## Hosted Version

The backend is hosted on [iban-gen.klump.solutions](https://iban-gen.klump.solutions) right now.

## API

```shell
curl https://iban-gen.duckdns.org/v1/random
```
or
```shell
curl https://iban-gen.duckdns.org/v1/random?bankCode=1009000
```
or
```shell
curl https://iban-gen.duckdns.org/v1/random?bic=BEVODEBBXXX
```
Check all available BICs with
```shell
curl https://iban-gen.duckdns.org/v1/bics
```
