FROM python:3.12-slim

WORKDIR /app

# Layer 1: deps (rarely change → cached across most builds)
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Layer 2: code + data (change together when a role/grant is added)
COPY bootstrap.py roles.yaml ./

ENTRYPOINT ["python", "bootstrap.py"]
CMD ["roles.yaml"]
