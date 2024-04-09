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
