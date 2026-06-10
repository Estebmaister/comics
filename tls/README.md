# Self-Signed Local TLS Certificate

The local frontend, Python backend, and Go backend use `tls/comics.crt` and
`tls/comics.key` for HTTPS development. Regenerate the certificate when the
browser reports `net::ERR_CERT_DATE_INVALID` or when the `notAfter` date is
near.

Run from the repository root:

```sh
openssl req -x509 -newkey rsa:2048 -nodes -sha256 \
  -keyout tls/comics.key \
  -out tls/comics.crt \
  -days 3650 \
  -config tls/server.cnf \
  -extensions v3_ext
```

This creates a 2048-bit RSA key and a self-signed certificate with the SANs
defined in `tls/server.cnf`, including `localhost` and `127.0.0.1`.

## Verify

```sh
openssl x509 -in tls/comics.crt -noout -subject -issuer -dates -serial
openssl x509 -in tls/comics.crt -noout -ext subjectAltName
```

## Trust on macOS

Remove any old trusted copy first, then add the regenerated certificate:

```sh
sudo security delete-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain tls/comics.crt

sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain tls/comics.crt
```

The delete command can fail if the old certificate is not trusted; that is safe
to ignore.

## Restart

Restart any running Python backend, Go backend, and frontend dev server after
replacing the certificate. Running processes keep using the certificate they
loaded at startup.

For local browser testing, use either `localhost` or `127.0.0.1` for the
frontend URL:

```text
https://localhost:3000/comics/
https://127.0.0.1:3000/comics/
```

The Python and Go server CORS configs allow local HTTP and HTTPS origins for
development.
