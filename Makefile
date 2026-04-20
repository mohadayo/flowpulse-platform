.PHONY: test test-python test-go test-ts lint up down build clean

test: test-python test-go test-ts
	@echo "All tests passed!"

test-python:
	cd services/event-collector && pip install -r requirements.txt -q && pytest -v

test-go:
	cd services/event-processor && go test -v ./...

test-ts:
	cd services/analytics-api && npm install --silent && npm test

lint: lint-python lint-go lint-ts
	@echo "All linters passed!"

lint-python:
	cd services/event-collector && flake8 --max-line-length=120 app.py

lint-go:
	cd services/event-processor && go vet ./...

lint-ts:
	cd services/analytics-api && npx eslint src/ --ext .ts

build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

clean:
	docker compose down -v --rmi local
