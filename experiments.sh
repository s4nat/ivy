#!/bin/bash

# Build and run the main program in the background
go build
./ivy &

# Wait for the program to initialize
sleep 2

# Start a client and send a writePage command
echo -e "1\nwritePage P1 Content1" | ./ivy
