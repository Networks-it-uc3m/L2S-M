#!/bin/sh
set -eu

target="${1:-10.0.0.1}"

nmap -sS "$target"
nmap -sN "$target"
nmap -sF "$target"
nmap -sX "$target"
