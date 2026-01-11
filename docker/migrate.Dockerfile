FROM migrate/migrate:v4.17.1

WORKDIR /migrations

COPY migrations/ ./
