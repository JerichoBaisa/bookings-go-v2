#!/bin/bash

go build -o bookings cmd/web/*.go
./bookings -dbname=bookings -dbuser=jericho -production=false -cache=false 