FROM python:3.11.6
RUN pip install kopf kubernetes PyMySQL cryptography requests
COPY ./src/operator/l2sm-operator.py /l2sm-operator.py
CMD kopf run --standalone --all-namespaces /l2sm-operator.py
