#!/bin/sh

# Create a redis database and run in the background
# Wait 1 second before starting Houston to ensure database is ready
redis-server & sleep 1

# Run houston CLI. Pass all arguments from 'docker run' to this command
houston "$@" &

# Wait for houston to finish. This ensures that this script terminates when there is a SIGTERM on the container
wait $!
