#!/bin/bash
set -e

# Set environment variables
export VERSION="2.2"
export DOCKERHUB_REPO="alexdecb"

# Function to build image
build_image() {
  local image_name="$1"
  local folder_name="$2"

  echo "Building ${image_name}..."
  docker build -t "${DOCKERHUB_REPO}/${image_name}:${VERSION}" -f "./build/${folder_name}/Dockerfile" .
}

# Function to push image
push_image() {
  local image_name="$1"

  echo "Pushing ${image_name}..."
  docker push "${DOCKERHUB_REPO}/${image_name}:${VERSION}"
}

# Option 1: Build image
if [ "$1" == "build" ]; then
  build_image "l2sm-switch" "switch"
  build_image "l2sm-controller" "controller"
  build_image "l2sm-operator" "operator"
  echo "Images have been built successfully."

# Option 2: Push image
elif [ "$1" == "push" ]; then
  push_image "l2sm-switch"
  push_image "l2sm-controller"
  push_image "l2sm-operator"
  echo "Images have been pushed successfully."

# Option 3: Build and push image
elif [ "$1" == "build_push" ]; then
  build_image "l2sm-switch" "switch"
  push_image "l2sm-switch"
  build_image "l2sm-controller" "controller"
  push_image "l2sm-controller"
  build_image "l2sm-operator" "operator"
  push_image "l2sm-operator"
  echo "Images have been built and pushed successfully."

# Invalid option
else
  echo "Invalid option. Please use 'build', 'push', or 'build_push'."
  exit 1
fi
