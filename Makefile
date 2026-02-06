.PHONY: build run docker-build docker-run clean

build:
	go build -o bin/lumo-api ./cmd/server

run: build
	./bin/lumo-api

docker-build:
	docker build -t lumo-api:latest .

docker-run:
	# Mount your local auth.json into the container (read‑only)
	docker run -d --name lumo-api \
	  -p 8080:8080 \
	  -v $(PWD)/auth.json:/app/auth.json:ro \
	  lumo-api:latest

clean:
	rm -rf bin
