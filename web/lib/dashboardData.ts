import type { Achievement, KPI } from "./api";

export type QuarterGroup = {
  quarter: string;
  year: string;
  items: KPI[];
};

export function groupKpisByQuarter(items: KPI[]): QuarterGroup[] {
  const grouped = new Map<string, KPI[]>();
  for (const item of items) {
    const bucket = grouped.get(item.quarter) ?? [];
    bucket.push(item);
    grouped.set(item.quarter, bucket);
  }

  return [...grouped.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([quarter, quarterItems]) => ({
      quarter,
      year: quarter.split("-")[0] ?? "",
      items: quarterItems,
    }));
}

export function quarterProgressForKpi(item: KPI): number {
  const code = item.quarter.split("-")[1]?.toUpperCase();
  switch (code) {
    case "Q1":
      return item.progressQ1;
    case "Q2":
      return item.progressQ2;
    case "Q3":
      return item.progressQ3;
    case "Q4":
      return item.progressQ4;
    default:
      return item.annualProgress;
  }
}

export function availableYears(groups: QuarterGroup[]): string[] {
  return [...new Set(groups.map((group) => group.year))].sort();
}

export function groupAchievementsByKpi(achievements: Achievement[], quarter: string): Map<number, Achievement[]> {
  const grouped = new Map<number, Achievement[]>();
  for (const item of achievements) {
    if (item.quarter !== quarter || item.kpiId == null) {
      continue;
    }
    const bucket = grouped.get(item.kpiId) ?? [];
    bucket.push(item);
    grouped.set(item.kpiId, bucket);
  }
  return grouped;
}

export function unmappedAchievements(achievements: Achievement[], quarter: string): Achievement[] {
  return achievements.filter((item) => item.quarter === quarter && item.kpiId == null);
}
