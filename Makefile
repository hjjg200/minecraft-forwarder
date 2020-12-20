
all: minecraft_forwarder

clean:
	rm -rf ./bin/

minecraft_forwarder:
	mkdir -p ./bin/
	go build -o ./bin/minecraft-forwarder ./cmd/minecraft-forwarder
