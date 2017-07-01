# Event Log Service


## Running locally with Docker

### Start a Postgres Container

```sh
docker pull postgres:9.6.3
docker run --name event-log-postgres -e POSTGRES_PASSWORD=pass1234 -d postgres:9.6.3
```

Test DB by starting the psql cli:

```sh
docker exec -ti event-log-postgres psql -U postgres
```

### Run migrations

```sh
docker pull tobyjsullivan/flyway:latest
docker run -ti -v `pwd`/db:/sql --link event-log-postgres:db flyway:latest -url=jdbc:postgresql://db:5432/postgres -user=postgres -password=pass1234 migrate
```

### Run event-log service

#### Create an `.env` file

You should start by creating a `.env` file with the necessary configurations.
This would be a good template:

```
PG_HOSTNAME=db
PG_USERNAME=postgres
PG_PASSWORD=pass1234
PG_DATABASE=postgres
```

#### Build and run the container

```sh
docker build -t event-log .
docker run -it --env-file=./.env --link event-log-postgres:db event-log
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
