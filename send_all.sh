#!/bin/bash

# Build Go project once
echo "Building SES project..."
go build -o ses.exe cmd/main.go
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Build success!"

# Start all processes in background with auto-send
for i in $(seq 0 14); do
    echo "Starting process $i..."
    ./ses.exe $i send > "logs/console_P$i.log" 2>&1 &
done

echo "All processes started!"
wait
