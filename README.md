# Backend

Go API server. Serves the REST API and the embedded React frontend from a single binary.

## Run locally

Make sure Postgres is running on `localhost:5432` and create a database:

```bash
createdb foodordering
```

Then start the server:

```bash
cd backend

export PORT=8080
export DATABASE_URL="postgres://localhost:5432/foodordering?sslmode=disable"
export JWT_SECRET="any-secret"
export REDIS_URL="localhost:6379"

go run ./cmd/server
```

Server starts on `http://localhost:8080`. Migrations and seed data run automatically on first start.

## Load promo codes into Redis (one-time)

```bash
cd backend

export REDIS_URL="localhost:6379"

go run ./cmd/coupon-worker
```

This downloads the 3 coupon gz files from S3 and loads them into Redis. Takes a couple of minutes. Once done, a `promo:ready` key is set in Redis.

## Build Docker image

Run this from the **root of the repo** (not inside `backend/`), as the root Dockerfile builds both frontend and backend together:

```bash
cd ..   # go to root

docker build -t food-ordering:v1.0.0 .
```

## Deploy code change to a running kind cluster

If you change Go code and need to push it to the local kind cluster:

```bash
# 1. rebuild the image (from repo root)
docker build -t food-ordering:v1.0.0 .

# 2. load it into kind
kind load docker-image food-ordering:v1.0.0 --name food-ordering

# 3. restart the deployment to pick up the new image
kubectl rollout restart deployment/food-ordering -n food-ordering

# 4. wait for it to finish
kubectl rollout status deployment/food-ordering -n food-ordering
```
