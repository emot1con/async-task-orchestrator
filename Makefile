SHELL := /bin/bash

up_build:
	docker-compose down
	docker-compose up -d --build

api:
	docker-compose down api
	docker-compose up -d api --build

worker:
	docker-compose down worker
	docker-compose up -d worker --build

app:
	docker-compose down worker api
	docker-compose up -d worker api --build