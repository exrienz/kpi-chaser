import test from "node:test";
import assert from "node:assert/strict";

import { groupAchievementsByKpi, groupKpisByQuarter, quarterProgressForKpi, unmappedAchievements } from "./dashboardData.ts";
import type { Achievement, KPI } from "./api.ts";

const kpis: KPI[] = [
  {
    id: 1,
    quarter: "2026-Q2",
    title: "Quarter Two",
    description: "",
    weight: 30,
    targetMetric: "",
    parentKpiId: null,
    progressQ1: 10,
    progressQ2: 60,
    progressQ3: 20,
    progressQ4: 30,
    annualProgress: 30,
  },
  {
    id: 2,
    quarter: "2026-Q1",
    title: "Quarter One",
    description: "",
    weight: 40,
    targetMetric: "",
    parentKpiId: null,
    progressQ1: 75,
    progressQ2: 20,
    progressQ3: 20,
    progressQ4: 20,
    annualProgress: 33,
  },
];

const achievements: Achievement[] = [
  {
    id: 1,
    quarter: "2026-Q1",
    rawText: "Achievement A",
    enhancedText: "",
    category: "",
    impactNote: "",
    status: "draft",
    kpiId: 2,
  },
  {
    id: 2,
    quarter: "2026-Q1",
    rawText: "Achievement B",
    enhancedText: "",
    category: "",
    impactNote: "",
    status: "draft",
    kpiId: null,
  },
];

test("groupKpisByQuarter sorts quarter buckets", () => {
  const grouped = groupKpisByQuarter(kpis);
  assert.deepEqual(grouped.map((item) => item.quarter), ["2026-Q1", "2026-Q2"]);
});

test("quarterProgressForKpi reads the matching quarterly field", () => {
  assert.equal(quarterProgressForKpi(kpis[0]), 60);
  assert.equal(quarterProgressForKpi(kpis[1]), 75);
});

test("achievement grouping keeps mapped and unmapped results separate", () => {
  const grouped = groupAchievementsByKpi(achievements, "2026-Q1");
  assert.equal(grouped.get(2)?.length, 1);
  assert.equal(unmappedAchievements(achievements, "2026-Q1").length, 1);
});
