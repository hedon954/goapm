#!/bin/bash

for i in {1..30}; do
  if mysqladmin ping -h 127.0.0.1 -P 3306 --silent; then
    echo "MySQL is up!"
    break
  fi
  echo "Waiting for MySQL..."
  sleep 1
done
