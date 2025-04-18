# printenv

printenv runs a http server that print environment variables.

```console
$ docker run -p 8080:8080 ghcr.io/fujiwara/printenv:v0.2.2
2021/08/13 08:34:01 starting up with local httpd :8080
```

```console
$ curl -s localhost:8080
HOME=/
HOSTNAME=e48a90ce50f8
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
PORT=8080

$ curl -s -H "accept: application/json" localhost:8080 | jq .
{
  "HOME": "/",
  "HOSTNAME": "a0e3875fed32",
  "PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
  "PORT": "8080"
}
```

## Show request headers

`/headers` shows request headers.

```console
$ curl -s localhost:8080/headers
Accept: */*
User-Agent: curl/7.68.0

$ curl -s -H "accept: application/json" localhost:8080/headers | jq .
{
  "Accept": "*/*"
  "User-Agent": "curl/7.68.0"
}
```

## Latency feature

You can add latency to the response.

Set CLI option `-latency` or environment variable `LATENCY` to add latency.

```console
- `-latency [time.Duration]` adds a latency into the response.
  - `LATENCY` environment variable can be used to set the latency.
- `-randomize` randomize latencies.
  - `RANDOMIZE` environment variable can be used to set the randomize.
```

By individual request, you can override latency by query parameter `latency`, for example: `?latency=1s`.

## LICENSE

MIT License
