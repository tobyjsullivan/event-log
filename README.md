# Event Log Service


## Running locally with Docker

### Start a Postgres Container

```sh
docker pull postgres:9.6.3
docker run --name event-log-postgres -e POSTGRES_PASSWORD=pass1234 -d postgres:9.6.3
```

Test the DB is running:

```sh
docker exec -ti event-log-postgres psql -U postgres
```

### Run migrations

```sh
docker pull tobyjsullivan/flyway:latest
docker run -ti --link event-log-postgres:db flyway:latest migrate -url=jdbc:postgresql://db:5432/postgres -user=postgres -password=pass1234
```

