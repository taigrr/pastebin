FROM golang:alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY . .

RUN go build -o /pastebin .
RUN go build -o /pb ./cmd/pb/


FROM alpine

EXPOSE 8000/tcp
ENTRYPOINT ["pastebin"]

COPY --from=builder /pastebin /bin/pastebin
COPY --from=builder /pb /bin/pb
