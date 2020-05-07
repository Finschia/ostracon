#!/usr/bin/env bash

echo "Terminating contract-test"
kill "$(lsof -i tcp:61322 | tail -n 1 | awk '{print $2}')"
