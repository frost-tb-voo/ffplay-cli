#!/bin/sh

DIR=$(cd $(dirname ${BASH_SOURCE:-$0}); pwd)
cd ${DIR}

sudo apt-get update
sudo apt-get install -y ffmpeg
dpkg --get-selections | grep codec

# go get github.com/gdamore/tcell
# go get github.com/nsf/termbox-go
go build .
mv cmd ffplay-cli
chmod +x ffplay-cli
