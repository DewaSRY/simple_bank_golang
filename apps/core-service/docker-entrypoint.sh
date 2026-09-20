#!/bin/sh
set -e

echo "running database migrations..."
./migration up

echo "starting server..."
exec ./main
