#!/bin/bash
set -e

# Set environment variables
export VERSION="2.5.1"
export DOCKERHUB_REPO="alexdecb"
export PLATFORMS="linux/amd64,linux/arm64"

# Function to build image for current platform
build_image_local() {
  local image_name="$1"
  local folder_name="$2"

  echo "Building ${image_name} for current platform..."
  docker build -t "${DOCKERHUB_REPO}/${image_name}:${VERSION}" -f "./build/${folder_name}/Dockerfile" .
}

# Function to push image
push_image() {
  local image_name="$1"

  echo "Pushing ${image_name}..."
  docker push "${DOCKERHUB_REPO}/${image_name}:${VERSION}"
}

# Function to build and push multi-architecture image
build_and_push_image_multiarch() {
  local image_name="$1"
  local folder_name="$2"

  echo "Building and pushing ${image_name} for platforms ${PLATFORMS}..."
  docker buildx build --platform "${PLATFORMS}" -t "${DOCKERHUB_REPO}/${image_name}:${VERSION}" -f "./build/${folder_name}/Dockerfile" --push .
}

# Option 1: Build image for current platform
if [ "$1" == "build" ]; then
  build_image_local "l2sm-switch" "switch"
  build_image_local "l2sm-controller" "controller"
  build_image_local "l2sm-operator" "operator"
  echo "Images have been built successfully."

# Option 2: Push image
elif [ "$1" == "push" ]; then
  push_image "l2sm-switch"
  push_image "l2sm-controller"
  push_image "l2sm-operator"
  echo "Images have been pushed successfully."

# Option 3: Build and push multi-architecture images
elif [ "$1" == "build_push" ]; then
  build_and_push_image_multiarch "l2sm-switch" "switch"
  # build_and_push_image_multiarch "l2sm-controller" "controller"
  build_and_push_image_multiarch "l2sm-operator" "operator"
  echo "Multi-architecture images have been built and pushed successfully."

# Invalid option
else
  echo "Invalid option. Please use 'build', 'push', or 'build_push'."
  exit 1
fi
