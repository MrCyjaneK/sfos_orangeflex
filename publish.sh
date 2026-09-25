#!/bin/sh
set -eu
cd "$(dirname "$0")"
sfosbuild 5.1.0.11 all ./apps/sfos/
gh release create "${1:?usage: $0 <tag>}" ./apps/sfos/rpms/*.rpm
