FROM golang:1.13

RUN apt-get update \
 && apt-get install -y ffmpeg \
 && apt-get autoclean -y \
 && apt-get autoremove -y \
 && rm -fr /var/lib/apt/lists
WORKDIR /ffplay-cli
ADD ./cmd/main.go /ffplay-cli/main.go
RUN go build .
CMD [/ffplay-cli/ffplay-cli]

