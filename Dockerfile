FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY bin/splitbill-api /app/splitbill-api

RUN mkdir -p /app/storage/public/images /app/storage/logs/general_log

EXPOSE 3000

CMD ["/app/splitbill-api"]
