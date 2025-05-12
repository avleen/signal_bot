FROM golang:1.24

WORKDIR /app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go/go.mod go/go.sum ./
RUN go mod download && go mod verify

COPY go/. .
RUN --mount=type=cache,target=/volume1/docker/cache go build -v -o /usr/local/bin/app

COPY common/prompt_summary.txt .
COPY common/chatbot_init_msg.txt .

CMD ["app"]
