<!---
 Copyright 2024  Universidad Carlos III de Madrid
 
 Licensed under the Apache License, Version 2.0 (the "License"); you may not
 use this file except in compliance with the License.  You may obtain a copy
 of the License at
 
   http://www.apache.org/licenses/LICENSE-2.0
 
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
 License for the specific language governing permissions and limitations under
 the License.
 
 SPDX-License-Identifier: Apache-2.0
-->

# Build Directory

This directory contains Dockerfiles and scripts for building and pushing Docker images for different components of the project. 

The files and scripts are meant to be run directly in the /L2S-M directory, as the COPY instructions will refer to the /L2S-M/src directory.

## Directory Structure:

- `./build/switch`: Dockerfile and related files for building the l2sm-switch Docker image.
- `./build/controller`: Dockerfile and related files for building the l2sm-controller Docker image.
- `./build/operator`: Dockerfile and related files for building the l2sm-operator Docker image.
- `./build/build_images.sh`: Bash script for automating the build and push process of Docker images.

## Script Usage:

### 1. Build Images:
```bash
./build/build_images.sh build
```

This command will build Docker images for l2sm-switch, l2sm-controller, and l2sm-operator.

### 2. Push Images:

```bash
./build/build_images.sh push
```

This command will push previously built Docker images to the specified DockerHub repository.

### 3. Build and Push Images:

```bash
./build/build_images.sh build_push
```

This command will both build and push Docker images.

Note: Make sure to set the appropriate environment variables in the script before running. (The repo name and the version tag)

For any additional details or customization, refer to the respective Dockerfiles and the build script.
