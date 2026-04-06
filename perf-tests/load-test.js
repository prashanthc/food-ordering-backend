import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

const errorRate = new Rate("errors");
const orderDuration = new Trend("order_duration");

export const options = {
  stages: [
    { duration: "30s", target: 10 },
    { duration: "1m",  target: 50 },
    { duration: "30s", target: 100 },
    { duration: "1m",  target: 100 },
    { duration: "30s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<500"],
    errors:            ["rate<0.01"],
  },
};

let authToken = "";

export function setup() {
  const email = `perftest_${Date.now()}@test.com`;

  const reg = http.post(
    `${BASE_URL}/auth/register`,
    JSON.stringify({ name: "Perf User", email, password: "perf1234" }),
    { headers: { "Content-Type": "application/json" } }
  );

  if (reg.status === 201) {
    return { token: reg.json("token") };
  }

  const login = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({ email, password: "perf1234" }),
    { headers: { "Content-Type": "application/json" } }
  );
  return { token: login.json("token") };
}

export default function (data) {
  const headers = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${data.token}`,
  };

  const productsRes = http.get(`${BASE_URL}/api/product`);
  check(productsRes, {
    "products status 200": (r) => r.status === 200,
    "products has items":  (r) => r.json().length > 0,
  });
  errorRate.add(productsRes.status !== 200);
  sleep(0.5);

  const categoryRes = http.get(`${BASE_URL}/api/product?category=Burger`);
  check(categoryRes, {
    "category filter 200": (r) => r.status === 200,
  });
  errorRate.add(categoryRes.status !== 200);
  sleep(0.3);

  const searchRes = http.get(`${BASE_URL}/api/product?search=pizza`);
  check(searchRes, {
    "search 200": (r) => r.status === 200,
  });
  errorRate.add(searchRes.status !== 200);
  sleep(0.3);

  const products = productsRes.json();
  if (products.length > 0) {
    const start = Date.now();
    const orderRes = http.post(
      `${BASE_URL}/api/order`,
      JSON.stringify({
        items: [
          { productId: products[0].id, quantity: 2 },
          { productId: products[1].id, quantity: 1 },
        ],
      }),
      { headers }
    );
    orderDuration.add(Date.now() - start);
    check(orderRes, {
      "order placed 200": (r) => r.status === 200,
      "order has id":     (r) => r.json("id") !== "",
    });
    errorRate.add(orderRes.status !== 200);
  }

  sleep(1);
}
