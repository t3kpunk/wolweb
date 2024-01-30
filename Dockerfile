# docker build -t wolweb .
FROM golang:1.21-alpine AS builder

LABEL org.label-schema.vcs-url="https://github.com/t3kpunk/wolweb" \
    org.label-schema.url="https://github.com/t3kpunk/wolweb/blob/master/README.md"

WORKDIR /app

# Install Dependecies
RUN apk update && apk upgrade && \
    apk add --no-cache git && \
    git clone https://github.com/t3kpunk/wolweb . && \
    go mod tidy && \
    go mod download

# Build Source Files
RUN go build -o wolweb .


# Create 2nd Stage final image
# -------- production --------
FROM alpine AS production
# Create app directory
WORKDIR /app


COPY --from=builder /app/index.html .
COPY --from=builder /app/wolweb .
COPY --from=builder /app/devices.json .
COPY --from=builder /app/config.json .
COPY --from=builder /app/static ./static

RUN apk add --no-cache curl

ARG WOLWEBPORT=8089
ENV WOLWEBPORT=${WOLWEBPORT}
EXPOSE ${WOLWEBPORT}

CMD ["/app/wolweb"]

EXPOSE ${WOLWEBPORT}
HEALTHCHECK --interval=5s --timeout=3s \
    CMD curl --silent --show-error --fail http://localhost:${WOLWEBPORT}/wolweb/health || exit 1
