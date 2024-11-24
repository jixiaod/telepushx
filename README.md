# TelePushX

Telepushx is a message push service that supports sending messages to multiple users for telegram bot.

## Features

- Send messages to multiple users at the same time
- Send messages limit rate control (Telegram API limit 30 messages per second),if send message too fast, will sleep a while
- Send messages limit time control, if push message not finished, will stop and wait for next time

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
nohup ./telepushx --port 3000 --log-dir ./logs 2&>1 &
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
