#!/bin/bash

if [ "$1" = "d" ]; then
    openssl aes-256-cbc -d -md sha512 -pbkdf2 -iter 1000000 -in comics.key.enc -out comics.key -pass pass:"$CA_KEY_PASSWORD"
else
    openssl aes-256-cbc -md sha512 -pbkdf2 -iter 1000000 -salt -in comics.key -out comics.key.enc -pass pass:"$CA_KEY_PASSWORD"
fi
