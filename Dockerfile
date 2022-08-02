FROM golang:1.18 as builder

WORKDIR /src

COPY . .

RUN go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o external-scaler cmd/main.go


FROM scratch

WORKDIR /

COPY --from=builder /src/external-scaler .

ENTRYPOINT ["/external-scaler"]
