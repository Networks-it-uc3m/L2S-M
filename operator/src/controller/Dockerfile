FROM onosproject/onos:2.7-latest

COPY . ./

RUN apt-get update && \
    apt-get install wget && \
    chmod +x ./setup_controller.sh

ENTRYPOINT ["./setup_controller.sh"]