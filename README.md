# TODO

## Build

```
docker build -t app-prod:latest -f Dockerfile.app .
docker build -t server-prod:latest -f Dockerfile.server .
```

## Run

```
docker run -d -p 3000:80 --name app app-prod:latest
docker run -d -p 8080:8080 --name server server-prod:latest
```

## Audio

```

```
