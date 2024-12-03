#!/bin/bash
set -e

# Set environment variables
export VERSION="1.1.1"
export DOCKERHUB_REPO="alexdecb"
export PLATFORMS="linux/amd64,linux/arm64"

# Function to build and push multi-architecture image
build_and_push_image_multiarch() {
  local image_name="$1"
  local folder_name="$2"

  echo "Building and pushing ${image_name} for platforms ${PLATFORMS}..."
  docker buildx build --platform "${PLATFORMS}" -t "${DOCKERHUB_REPO}/${image_name}:${VERSION}" -f "./build/${folder_name}/Dockerfile" --push .
}
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
  build_image "lpm-collector" "collector"
  build_image "lpm-exporter" "swm_exporter"

  echo "Images have been built successfully."

# Option 2: Push image
elif [ "$1" == "push" ]; then
  push_image "lpm-collector"
  push_image "lpm-exporter"

  echo "Images have been pushed successfully."

# Option 3: Build and push image
elif [ "$1" == "build_push" ]; then
  build_and_push_image_multiarch "lpm-collector" "collector"
  build_and_push_image_multiarch "lpm-exporter" "swm_exporter"
  echo "Images have been built and pushed successfully."

# Invalid option
else
  echo "Invalid option. Please use 'build', 'push', or 'build_push'."
  exit 1
fi
