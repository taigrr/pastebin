# pastebin

[![Build Status](https://cloud.drone.io/api/badges/taigrr/pastebin/status.svg)](https://cloud.drone.io/taigrr/pastebin)
[![CodeCov](https://codecov.io/gh/taigrr/pastebin/branch/master/graph/badge.svg)](https://codecov.io/gh/taigrr/pastebin)
[![Go Report Card](https://goreportcard.com/badge/taigrr/pastebin)](https://goreportcard.com/report/taigrr/pastebin)
[![GoDoc](https://godoc.org/github.com/taigrr/pastebin?status.svg)](https://godoc.org/github.com/taigrr/pastebin)

pastebin is a self-hosted pastebin web app that lets you create and share
"ephemeral" data between devices and users. There is a configurable expiry
(TTL) after which the paste expires and is purged. There is also a handy
CLI for interacting with the service in a easy way or you can also use curl!

### Source

```bash
go install github.com/taigrr/pastebin@latest
go install github.com/taigrr/pastebin/cmd/pb@latest
```

## Usage

Run pastebin:

```bash
pastebin
```

Create a paste:

```bash
echo "Hello World" | pb
http://localhost:8000/p/92sHUeGPfo
```

Or use the Web UI: http://localhost:8000/

Or curl:

```bash
echo "Hello World" | curl -q -L --form blob='<-' -o - http://localhost:8000/
```

## Configuration

When running the `pastebin` server there are a few default options you might
want to tweak:

```
$ pastebin --help
  ...
  --expiry duration   expiry time for pastes (default 5m0s)
  --fqdn string      FQDN for public access (default "localhost")
  --bind string       address and port to bind to (default "0.0.0.0:8000")
```

Setting a custom `--expiry` lets you change when pastes are automatically
expired (*the purge time is 2x this value*). The `--fqdn` option is used as
a namespace for the service.

The command-line utility by default talks to http://localhost:8000 which can
be changed via the `--url` flag:

```bash
pb --url https://paste.mydomain.com/
```

## License

MIT
