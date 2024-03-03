#!/bin/bash
sudo apt update
sudo apt install -y go-golang git wget

wget https://www.example.com/

git clone https://github.com/DeaSTL/flappybird-htmx

cd flappybird-htmx

go run ./
