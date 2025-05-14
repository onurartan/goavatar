#!/bin/bash

set -e

echo "🔧 Building the project..."
go build -o app ./src

echo "🚀 Running the project..."
go run  ./src
