FROM python:3.12-slim

WORKDIR /app

# Layer 1: deps (rarely change → cached across most builds)
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Layer 2: code + data (change together when the seeded layout changes)
COPY bootstrap.py dashboard.yaml ./

ENTRYPOINT ["python", "bootstrap.py"]
CMD ["dashboard.yaml"]
