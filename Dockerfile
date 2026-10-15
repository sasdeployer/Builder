FROM golang:1.26.9 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/builder .

FROM alpine:3.22
RUN apk add --no-cache git
COPY --from=build /out/builder /usr/local/bin/builder
ENTRYPOINT ["builder"]

# docker build -t builder .
# docker run --rm -v "$PWD":/work -w /work builder init <repo url>
