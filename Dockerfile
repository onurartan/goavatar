FROM golang:1.24-alpine


RUN apk add --no-cache \
    build-base \
    fontconfig \
    freetype \
    ttf-dejavu \
    git \
    dos2unix

WORKDIR /app


COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o app ./src

EXPOSE 9980

CMD ["./app"]
