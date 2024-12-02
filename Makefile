APP_NAME=example-be
DEFAULT_PORT=8100
.PHONY: setup init build dev test db-migrate-up db-migrate-down

setup:
	cd ~ && go get -v github.com/rubenv/sql-migrate/...
	cd ~ && go get github.com/golang/mock/gomock
	cd ~ && go get github.com/golang/mock/mockgen
	cp .env.sample .env && vim .env


dev:
	go run ./cmd/server/main.go

docker-build:
	docker build \
	--build-arg DEFAULT_PORT="${DEFAULT_PORT}" \
	-t ${APP_NAME}:latest .

build:
	sudo docker build -t truongvanhuy2000/bloom-be .

push:
	sudo docker push truongvanhuy2000/bloom-be