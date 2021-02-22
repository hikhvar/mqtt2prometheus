#!/bin/sh

user=mqtt2prometheus
if ! getent passwd "${user}" > /dev/null; then
    useradd --system --home-dir /var/lib/${user} --no-create-home || true
fi