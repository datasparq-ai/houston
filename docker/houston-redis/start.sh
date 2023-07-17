#!/bin/sh

# Container interrupt handler. Before stopping the container, let Redis save a dump.rdb file, then stop all processes.
function sigint_handler()
{
    kill $houston_pid
    kill $redis_pid
    wait $redis_pid
    exit 1
}
trap sigint_handler SIGINT

# Create a redis database and run in the background
redis-server &
redis_pid=$!

# Wait 1 second before starting Houston to ensure database is ready
sleep 1

# Run houston CLI. Pass all arguments from 'docker run' to this command
houston "$@" &
houston_pid=$!

# Wait for houston to finish
wait $houston_pid
