import http from "k6/http";
import { sleep } from "k6";
import { Counter } from "k6/metrics";

// Custom metrics
const flowsStarted = new Counter("flows_started");
const flowsCompleted = new Counter("flows_completed");
const flowsFailed = new Counter("flows_failed");
const errorRate = new Rate("error_rate");

const ENGINE_URL = __ENV.ENGINE_URL || "http://localhost:8080";

export let options = {
  // Options can be overridden via CLI flags (--vus, --duration, etc.)
};

export function setup() {
  // Register simple step
  const step = {
    id: "k6-simple-step",
    name: "K6 Simple Step",
    type: "script",
    attributes: {
      input: { role: "required", type: "string" },
      result: { role: "output", type: "string" },
    },
    script: {
      language: "ale",
      script: '{:result "hello"}',
    },
  };

  const res = http.post(`${ENGINE_URL}/engine/step`, JSON.stringify(step), {
    headers: { "Content-Type": "application/json" },
  });

  if (res.status !== 201 && res.status !== 409) {
    throw new Error(`Failed to register step: ${res.status}`);
  }

  console.log("Step registered");
  return { stepId: step.id };
}

export default function (data) {
  const flowId = `k6-${__VU}-${__ITER}-${Date.now()}`;

  // Start flow
  const startRes = http.post(
    `${ENGINE_URL}/engine/flow`,
    JSON.stringify({
      id: flowId,
      goals: [data.stepId],
      init: { input: "test" },
    }),
    {
      headers: { "Content-Type": "application/json" },
      tags: { name: "StartFlow" },
    }
  );

  if (startRes.status !== 201) {
    flowsFailed.add(1);
    errorRate.add(1);
    return;
  }

  flowsStarted.add(1);

  // Poll for completion (max 5 seconds)
  let completed = false;
  let attempts = 0;
  const maxAttempts = 50;

  while (!completed && attempts < maxAttempts) {
    sleep(0.1);

    const statusRes = http.get(`${ENGINE_URL}/engine/flow/${flowId}`, {
      tags: { name: "GetFlowStatus" },
    });

    if (statusRes.status === 200) {
      const flow = JSON.parse(statusRes.body);
      if (flow.status === "completed") {
        completed = true;
        flowsCompleted.add(1);
        errorRate.add(0);
      } else if (flow.status === "failed") {
        flowsFailed.add(1);
        errorRate.add(1);
        return;
      }
    }
    attempts++;
  }

  if (!completed) {
    flowsFailed.add(1);
    errorRate.add(1);
  }
}

export function handleSummary(data) {
  const duration = (data.state.testRunDurationMs || 0) / 1000;
  const started = data.metrics.flows_started?.values?.count || 0;
  const completed = data.metrics.flows_completed?.values?.count || 0;
  const failed = data.metrics.flows_failed?.values?.count || 0;
  const errorPct = data.metrics.error_rate?.values?.rate || 0;
  const maxVUs = data.metrics.vus_max?.values?.max || 0;

  const throughput = completed / duration;

  console.log("\n=== RESULTS ===");
  console.log(`Duration:        ${duration.toFixed(1)}s`);
  console.log(`VUs:             ${maxVUs}`);
  console.log(`Started:         ${started}`);
  console.log(`Completed:       ${completed}`);
  console.log(`Failed:          ${failed}`);
  console.log(`Throughput:      ${throughput.toFixed(1)} flows/sec`);
  console.log(`Error Rate:      ${(errorPct * 100).toFixed(2)}%`);
  console.log(`Success Rate:    ${((1 - errorPct) * 100).toFixed(2)}%`);

  return {};
}
