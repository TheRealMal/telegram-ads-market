#!/bin/bash

# Name and label of the devopment container
name=ads-mrkt-dev

# Name of the network used by all containers
shared_network=ads-mrkt-network

# Absolute path to the directory containing this script
self_dir="$(dirname $(realpath $0))"

# Location of the cache that will hold all go packages downloaded inside the
# dev container
go_pkg_cache_dir="$self_dir/.go_pkg_cache"
# Location of the Go build cache
go_build_cache_dir="$self_dir/.go_build_cache"

# Set the BOT_TOKEN env var
bot_token=${BOT_TOKEN:-fakedevtoken}
# Set proxy port
proxy_port=""

# Ensure .env exists from example.env when running app commands
ensure_env() {
    if [[ ! -f "$self_dir/.env" ]]; then
        if [[ -f "$self_dir/example.env" ]]; then
            cp "$self_dir/example.env" "$self_dir/.env"
            echo "[$name] Created .env from example.env — please edit .env with your secrets"
        else
            echo "[$name] No .env or example.env found"
            exit 1
        fi
    fi
}

# Generate the usage message by parsing comments of this file
print_usage() {
    echo -e "Usage: $(realpath $0) <command>\n"
    echo -e "Available commands:\n"
    sed -n 's/\s\+#?//p' "$(realpath $0)" | column -t -s ':'
}

case "${1}" in
    #? build: Build the dev container and initialize/create dependencies
    build)
        # Create .env from example.env if missing
        if [[ ! -f "$self_dir/.env" && -f "$self_dir/example.env" ]]; then
            cp "$self_dir/example.env" "$self_dir/.env"
            echo "[$name] Created .env from example.env — please edit .env with your secrets"
        fi

        # Attempt to find the shared network in list of active docker networks
        network=$(docker network ls --format "{{.Name}}" \
            | grep -w "$shared_network")

        # Create the Go cache dirs

        if [[ ! -d $go_pkg_cache_dir ]]; then
            mkdir $go_pkg_cache_dir
            echo "[$name] Created Go pkg cache dir at: $go_pkg_cache_dir"
        fi
        chmod 777 "$go_pkg_cache_dir"

        if [[ ! -d $go_build_cache_dir ]]; then
            mkdir $go_build_cache_dir
            echo "[$name] Created Go build cache dir at: $go_build_cache_dir"
        fi
        chmod 777 "$go_build_cache_dir"

        # Create the network if it doesn't exist
        if [ "$network" = "" ]; then
            echo "[$name] Creating external network: $shared_network"
            docker network create \
                --gateway 172.32.1.1 \
                --ip-range 172.32.1.0/24 \
                --subnet 172.32.0.0/16 \
                "$shared_network"
        fi

        # Build the container
        echo "[$name] Building dev container"
        docker build -t "$name" --build-arg NAME="$name" ./dockerfiles/dev
	;;

    #? run-all: Run bot, market and userbot in the background
    run-all)
        ensure_env
        export ENV_FILE=.env
        go run ./cmd/main.go userbot run &
        LOCAL_PORT_PREFIX=1000 go run ./cmd/main.go market http &
        LOCAL_PORT_PREFIX=2000 go run ./cmd/main.go bot http &
        go run ./cmd/proxy/main.go &
        wait
    ;;

    #? run-bot: Run the Telegram bot service
    run-bot)
        ensure_env
        ENV_FILE=.env LOCAL_PORT_PREFIX=2000 go run ./cmd/main.go bot http
    ;;

    #? run-market: Run the market API service
    run-market)
        ensure_env
        ENV_FILE=.env LOCAL_PORT_PREFIX=1000 go run ./cmd/main.go market http
    ;;

    #? run-userbot: Run the user bot (Telegram client) service
    run-userbot)
        ensure_env
        ENV_FILE=.env go run ./cmd/main.go userbot run
    ;;

    #? start: Start the dev container
    start)
        # Attempt to find the shared network in list of active docker networks
        network=$(docker network ls --format "{{.Name}}" \
            | grep -w "$shared_network")

        ip=""

        # Notify the user and set the $network variable acoordingly
        if [ "$network" = "" ]; then
            echo "[$name] No external network"
        else
            echo "[$name] Connected to: $network"
            network="--network $network"
            ip="--ip 172.32.0.69"
        fi

        # Pick the startup command to run based on the mode selected

        docker run -it --rm \
            -v `pwd`:/app \
            -v "$go_pkg_cache_dir":/go/pkg \
            -v "$go_build_cache_dir":/app/.go_build_cache \
            -e GOCACHE=/app/.go_build_cache \
            -p 8080:8080 \
            -u not \
            -l "$name" \
            --name "$name" \
            $network $ip \
            "$name" /bin/bash
	;;

    #? stop: Stop the dev container
    stop)
        docker stop "$name"
    ;;

    #? shell: Run a shell session in the running dev container
    shell)
        # Attempt to find the id of the current dev instance
        instance=$(docker ps -f "label=$name" -q)

        # Run container
        docker exec -ti "$instance" /bin/bash
	;;

    #? help|-h|--help: Print this message
    help|-h|--help)
        print_usage;;

    *)
        print_usage
        echo "Invalid argument: $1"

        exit 1;;
esac
