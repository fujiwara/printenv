# printenv

printenv runs a http server that print environment variables.

```console
$ docker run -p 8080:8080 ghcr.io/fujiwara/printenv:v0.0.2
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
