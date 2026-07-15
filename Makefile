.PHONY: build run test test-cover test-race test-verbose dev-frontend dev-backend clean

build:
	cd backend && CGO_ENABLED=1 go build -o ../server ./cmd/main.go

run:
	cd backend && go run ./cmd/main.go

test:
	cd backend && CGO_ENABLED=1 go test ./... -count=1

test-cover:
	cd backend && CGO_ENABLED=1 go test ./... -coverprofile=coverage.out -covermode=atomic
	cd backend && go tool cover -html=coverage.out -o ../coverage.html

test-race:
	cd backend && CGO_ENABLED=1 go test -race ./... -count=1

test-verbose:
	cd backend && CGO_ENABLED=1 go test -v ./... -count=1

dev-frontend:
	cd frontend && npm run dev

dev-backend:
	cd backend && go run ./cmd/main.go

clean:
	rm -f server
	rm -rf backend/ui/dist
	rm -rf frontend/dist
