# telepushx

Telepushx is a Telegram message push service that supports sending messages to multiple users at the same time.

## Features

- Send messages to multiple users at the same time
- Send messages to users in batches
- Send messages to users in real-time

## Usage


## Configuration

```bash
cp .env.example .env
```

## Run

```bash
go run main.go
```     

## Build

```bash
go build -ldflags "-s -w" -o telepushx 
```

```bash
./telepushx -help
./telepushx --port 3000 --log-dir ./logs 
```


## License

[MIT](./LICENSE)