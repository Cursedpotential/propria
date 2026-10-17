FROM python:3.13-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY app.py domains.py response_projection.py domain_registry.json ./
COPY think-library ./think-library
COPY static ./static
USER 65534:65534
EXPOSE 8790
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8790", "--no-access-log"]
