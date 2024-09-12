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

## API 

### Send Message

```bash
curl -X POST http://localhost:3000/api/push/1
```
### Send Preview Message

```bash
curl -X POST http://localhost:3000/api/preview/1/1234567890
```
