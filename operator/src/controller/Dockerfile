FROM onosproject/onos:2.7.0

COPY . ./

RUN chmod +x ./setup_controller.sh

ENTRYPOINT ["./setup_controller.sh"]