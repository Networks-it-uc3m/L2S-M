FROM onosproject/onos:2.7-latest


RUN apt-get update && \
    apt-get install -y wget ssh sshpass
    
COPY ./src/controller ./

RUN chmod +x ./setup_controller.sh && \
    chmod +x ./onos_critique.sh

ENTRYPOINT ["./setup_controller.sh"]