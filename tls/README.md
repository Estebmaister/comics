# Self-signed certificate generation

Generate a 2048-bit RSA private key and save it, and the self-signed certificate and save it.
```sh
openssl req -newkey rsa:2048 -keyout comics.key \
-x509 -out comics.crt -days 365 -nodes -sha256 \
-config server.cnf -extensions v3_ext
```

Generate only the certificate with the existing key
```sh
openssl req -new -key comics.key -out server.csr -config server.cnf

openssl x509 -req -days 365 -in server.csr -signkey comics.key -CAcreateserial \
-out comics.crt -days 365 -sha256 -extfile server.cnf -extensions v3_ext
```

## Optional: to view the content of the certificate, private key or CSR
```sh
# Create a CSR using the private key
openssl req -new -key comics.key -out server.csr -config server.cnf
# View the content of the CSR and verify the x509v3 section with the SAN entered in server.cnf
openssl req -noout -text -in server.csr

# View the content of the certificate
openssl x509 -in comics.crt -text -noout

# View the content of the private key
openssl rsa -in comics.key -text -noout
```

## Optional: add/remove certificate to MacOS keychain
```sh
# Add it to the keychain
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain comics.crt

# Remove it from the keychain
sudo security delete-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain comics.crt
```