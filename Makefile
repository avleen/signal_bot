.EXPORT_ALL_VARIABLES:

DOCKER_BUILDKIT = 1

all: build run

build:
	@docker build -t signal_bot:latest .

run:
	@cd /volume1/docker
	@docker-compose up -d --build signal_bot
