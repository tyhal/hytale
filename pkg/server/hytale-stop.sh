#!/bin/sh

FIFO="/run/hytale/stdin.fifo"

if [ -p "$FIFO" ]; then
  echo "say [Server] Server shutting down in 60 seconds..." > "$FIFO"
  sleep 30
  echo "say [Server] Server shutting down in 30 seconds..." > "$FIFO"
  sleep 20
  echo "say [Server] Server shutting down in 10 seconds..." > "$FIFO"
  sleep 10

  echo "say [Server] World backup initiated..." > "$FIFO"
  echo "backup" > "$FIFO"
  echo "say [Server] goodbye" > "$FIFO"
  sleep 5

  echo "stop" > "$FIFO"
fi
