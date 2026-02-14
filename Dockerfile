FROM golang:1.25-alpine as build

WORKDIR /app
COPY . .
RUN go build -o /budget-server

FROM golang:1.25-alpine as build-sql

WORKDIR /build
COPY . .
RUN go build -o budget-db-migration scripts/db_migrations/main.go

FROM alpine

COPY --from=build /budget-server /budget-server
COPY --from=build-sql /build/scripts/db_migrations/migrations /migrations
COPY --from=build-sql /build/budget-db-migration /budget-db-migration

CMD /budget-db-migration && /budget-server
