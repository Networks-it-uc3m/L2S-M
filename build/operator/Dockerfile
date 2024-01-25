FROM python:3.11.6

WORKDIR /usr/src/app

COPY ./src/operator/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY ./src/operator/l2sm-operator.py .

CMD kopf run --liveness=http://0.0.0.0:8080/healthz --standalone --all-namespaces ./l2sm-operator.py
