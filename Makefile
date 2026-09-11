.PHONY: build test race vet verify tmux-test real-test fuzz install

build:
	go build -trimpath -o bin/aiswitch ./cmd/aiswitch

test:
	go test ./...

race:
	go test -race -coverprofile=coverage.out ./...

vet:
	go vet ./...

verify: vet race build

tmux-test:
	AISWITCH_TEST_TMUX=1 go test ./integration -run TestTmux -v -count=1 -timeout=60s

real-test:
	AISWITCH_TEST_REAL=1 go test ./integration -run TestReal -v -count=1 -timeout=180s

fuzz:
	go test ./internal/profile -run '^$$' -fuzz FuzzValidateName -fuzztime=5s

install:
	mkdir -p "$(HOME)/.local/bin"
	go build -trimpath -o "$(HOME)/.local/bin/aiswitch" ./cmd/aiswitch
