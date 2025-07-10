// k6_load_test.js - k6 script for load and chaos testing.
import http from "k6/http";
import { sleep, check } from "k6";

export let options = {
  stages: [
    { duration: "2m", target: 100 },
    { duration: "5m", target: 100 },
    { duration: "2m", target: 0 }
  ],
  thresholds: {
    http_req_duration: ["p(95)<200"]
  }
};

export default function () {
  let res = http.get("https://api.iaros.ai/healthcheck");
  check(res, { "status is 200": (r) => r.status === 200 });
  sleep(1);
}
