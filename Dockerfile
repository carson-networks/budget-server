FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG TARGETOS TARGETARCH

WORKDIR /app
COPY . .
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN go build -mod=vendor -o /budget-server \
    && go build -mod=vendor -o /budget-db-migration scripts/db_migrations/main.go

FROM alpine

COPY --from=build /budget-server /budget-server
COPY --from=build /app/scripts/db_migrations/migrations /migrations
COPY --from=build /budget-db-migration /budget-db-migration

CMD /budget-db-migration && /budget-server
