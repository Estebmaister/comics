#!/bin/bash

if [ "$1" = "d" ]; then
    openssl aes-256-cbc -md sha512 -pbkdf2 -iter 1000000 \
    -pass pass:"$CA_KEY_PASSWORD" -salt -in comics.key.enc -out comics.key -d 
else
    openssl aes-256-cbc -md sha512 -pbkdf2 -iter 1000000 \
    -pass pass:"$CA_KEY_PASSWORD" -salt -in comics.key -out comics.key.enc
fi
