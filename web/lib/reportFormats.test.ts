import test from "node:test";
import assert from "node:assert/strict";

import { formatReport } from "./reportFormats.ts";

const report = {
  id: 1,
  quarter: "2026-Q1",
  title: "Quarterly KPI Summary 2026-Q1",
  body: "Improve security posture\n- Reduced attack surface\n\nAutomate response workflows\n- Built phishing triage automation",
};

test("formats default review output without changes", () => {
  assert.equal(formatReport(report, "review"), report.body);
});

test("formats notion output with heading and bullet conversion", () => {
  const output = formatReport(report, "notion");
  assert.match(output, /^# Quarterly KPI Summary 2026-Q1/);
  assert.match(output, /\* Reduced attack surface/);
});

test("formats hr output with metadata prefix", () => {
  const output = formatReport(report, "hr");
  assert.match(output, /Quarter: 2026-Q1/);
  assert.match(output, /Document: Quarterly KPI Summary 2026-Q1/);
});

test("returns empty-state copy when report is missing", () => {
  assert.equal(formatReport(null, "review"), "No report generated yet.");
});
