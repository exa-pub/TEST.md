.PHONY: build test sync-skill clean

build:
	go build -o ./bin/ ./cmd/...

test:
	go test ./internal/...

sync-skill:
	mkdir -p skills/testmd/references
	cp docs/specification.md skills/testmd/references/specification.md
	cp docs/cli.md skills/testmd/references/cli.md
	cp docs/examples.md skills/testmd/references/examples.md
	cp docs/architecture.md skills/testmd/references/architecture.md

clean:
	rm -rf ./bin/

all: build test sync-skill
