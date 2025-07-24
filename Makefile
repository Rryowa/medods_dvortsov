run:
	go run ./cmd/main.go

restart: down up

down:
	docker compose down -v

up:
	docker compose up -d --build
	docker compose logs -f app

lint:
	golangci-lint run ./... --fix

gen:
	go generate ./...

webhook:
	go run webhook_receiver.go

keygen:
	@KEY=$$(openssl rand -base64 32); \
	if grep -q '^JWT_SECRET=' .env; then \
		sed -i.bak "s#^JWT_SECRET=.*#JWT_SECRET=$$KEY#" .env && rm .env.bak; \
	else \
		echo -n "\nJWT_SECRET=$$KEY" >> .env; \
	fi
	@KEY=$$(openssl rand -base64 32); \
	if grep -q '^AUTH_SERVICE_API_KEY=' .env; then \
		sed -i.bak "s#^AUTH_SERVICE_API_KEY=.*#AUTH_SERVICE_API_KEY=$$KEY#" .env && rm .env.bak; \
	else \
		echo -n "\nAUTH_SERVICE_API_KEY=$$KEY" >> .env; \
	fi

.PHONY: run restart up down lint gen webhook keygen