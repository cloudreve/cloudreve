#!/bin/bash
set -e
export NODE_OPTIONS="--max-old-space-size=8192"

# This script is used to build the assets for the application.
cd frontend
rm -rf build
yarn install --network-timeout 1000000
yarn version --new-version $1 --no-git-tag-version
yarn run build

# Copy the build files to the application directory.
# The embed expects an "assets/build/" prefix inside the zip — stage it under that name.
cd ../
mkdir -p assets
rm -rf assets/build
cp -r frontend/build assets/build
zip -r - assets/build >assets.zip
rm -rf assets
mv assets.zip application/statics