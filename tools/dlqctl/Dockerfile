FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY dlqctl.py ./
ENV OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317 \
    OTEL_SERVICE_NAME=dlqctl \
    PYTHONUNBUFFERED=1
ENTRYPOINT ["python","/app/dlqctl.py"]

