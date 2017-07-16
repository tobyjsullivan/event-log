# Event Log Service

## Running locally with docker-compose

### Copy and configure .env files

```sh
cp ./env/sample/*.env ./env 
# Edit all .env files as needed
```

### Start the stack

```sh
docker-compose up
# Note: You may need to do this twice to properly configure DB.
```

### (Optional) Connect to the postgres instance

```sh
docker-compose run db psql -h db -U postgres
```

## API

The service supports the following commands:

### POST /commands/create-log

Params:
- log-id (UUID or hex-encoded 16-byte array)

### POST /commands/append-event

Params:
- log-id (UUID or hex-encoded 16-byte array)
- event-type (string)
- event-data (base64-encoded byte string)
