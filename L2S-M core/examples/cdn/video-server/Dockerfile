#-------------------------------------------------------------------------------
# Copyright 2024  Universidad Carlos III de Madrid
# 
# Licensed under the Apache License, Version 2.0 (the "License"); you may not
# use this file except in compliance with the License.  You may obtain a copy
# of the License at
# 
#   http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
# License for the specific language governing permissions and limitations under
# the License.
# 
# SPDX-License-Identifier: Apache-2.0
#-------------------------------------------------------------------------------

# Use the official Nginx image as the base image
FROM nginx:latest

# Set the working directory to /usr/share/nginx/html
WORKDIR /usr/share/nginx/html

# Copy the video file into the container
COPY big_buck_bunny.avi .

# Create an Nginx configuration file to serve the video
RUN echo "server {" > /etc/nginx/conf.d/default.conf \
    && echo "    listen 10.0.2.2:80;" >> /etc/nginx/conf.d/default.conf \
    && echo "    location / {" >> /etc/nginx/conf.d/default.conf \
    && echo "        root /usr/share/nginx/html;" >> /etc/nginx/conf.d/default.conf \
    && echo "        index big_buck_bunny.avi;" >> /etc/nginx/conf.d/default.conf \
    && echo "        autoindex on;" >> /etc/nginx/conf.d/default.conf \
    && echo "        types {" >> /etc/nginx/conf.d/default.conf \
    && echo "            video/avi avi;" >> /etc/nginx/conf.d/default.conf \
    && echo "        }" >> /etc/nginx/conf.d/default.conf \
    && echo "    }" >> /etc/nginx/conf.d/default.conf \
    && echo "}" >> /etc/nginx/conf.d/default.conf

RUN apt update && apt install -y iproute2
# Sleep indefinitely to keep the container running
CMD ["sleep", "infinity"]
