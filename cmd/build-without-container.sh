#!/bin/sh

sudo apt-get update
sudo apt-get install -y ffmpeg
dpkg --get-selections | grep codec

go build .

