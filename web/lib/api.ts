export type KPI = {
  id: number;
  userId?: number;
  quarter: string;
  title: string;
  description: string;
  weight: number;
  targetMetric: string;
  parentKpiId?: number | null;
  progressQ1: number;
  progressQ2: number;
  progressQ3: number;
  progressQ4: number;
  annualProgress: number;
  children?: KPI[];
};

export type Achievement = {
  id: number;
  quarter: string;
  rawText: string;
  enhancedText: string;
  category: string;
  impactNote: string;
  status: string;
  kpiId?: number | null;
};

export type Report = {
  id: number;
  quarter: string;
  title: string;
  body: string;
};

export type DashboardSummary = {
  quarter: string;
  totalKpis: number;
  totalAchievements: number;
  enhancedAchievements: number;
  mappedAchievements: number;
  draftAchievements: number;
  pendingJobs: number;
  kpiProgress: Array<{
    kpiId: number;
    title: string;
    weight: number;
    achievementCount: number;
    enhancedCount: number;
    progressPercent: number;
  }>;
};

const API_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api";

export class APIError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit, token?: string): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init?.headers ?? {}),
    },
    cache: "no-store",
  });

  if (!response.ok) {
    const data = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new APIError(data?.error ?? `Request failed with ${response.status}`, response.status);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}

export type CreateKPIInput = {
  quarter: string;
  title: string;
  description: string;
  weight: number;
  targetMetric: string;
  parentKpiId?: number | null;
  progressQ1?: number;
  progressQ2?: number;
  progressQ3?: number;
  progressQ4?: number;
};

export type UpdateKPIInput = CreateKPIInput;

export type UpdateKPIProgressInput = {
  progressQ1?: number;
  progressQ2?: number;
  progressQ3?: number;
  progressQ4?: number;
};

function quarterQuery(quarter: string): string {
  return quarter ? `?quarter=${encodeURIComponent(quarter)}` : "";
}

export function listKPIs(token: string, quarter: string): Promise<KPI[]> {
  return apiFetch<KPI[]>(`/kpis${quarterQuery(quarter)}`, undefined, token);
}

export function listKPIHierarchy(token: string, quarter: string): Promise<KPI[]> {
  return apiFetch<KPI[]>(`/kpis/hierarchy${quarterQuery(quarter)}`, undefined, token);
}

export function getKPIChildren(token: string, parentID: number): Promise<KPI[]> {
  return apiFetch<KPI[]>(`/kpis/${parentID}/children`, undefined, token);
}

export function createKPI(token: string, input: CreateKPIInput): Promise<KPI> {
  return apiFetch<KPI>(
    "/kpis",
    {
      method: "POST",
      body: JSON.stringify(input),
    },
    token,
  );
}

export function createSubKPI(token: string, parentID: number, input: Omit<CreateKPIInput, "quarter" | "parentKpiId">): Promise<KPI> {
  return apiFetch<KPI>(
    `/kpis/${parentID}/subkpis`,
    {
      method: "POST",
      body: JSON.stringify(input),
    },
    token,
  );
}

export function updateKPI(token: string, id: number, input: UpdateKPIInput): Promise<KPI> {
  return apiFetch<KPI>(
    `/kpis/${id}`,
    {
      method: "PUT",
      body: JSON.stringify(input),
    },
    token,
  );
}

export function updateKPIProgress(token: string, id: number, input: UpdateKPIProgressInput): Promise<KPI> {
  return apiFetch<KPI>(
    `/kpis/${id}/progress`,
    {
      method: "PUT",
      body: JSON.stringify(input),
    },
    token,
  );
}

export function deleteKPI(token: string, id: number): Promise<void> {
  return apiFetch<void>(`/kpis/${id}`, { method: "DELETE" }, token);
}

export function getDashboardSummary(token: string, quarter: string): Promise<DashboardSummary> {
  return apiFetch<DashboardSummary>(`/dashboard${quarterQuery(quarter)}`, undefined, token);
}

export function listAchievements(token: string, quarter: string): Promise<Achievement[]> {
  return apiFetch<Achievement[]>(`/achievements${quarterQuery(quarter)}`, undefined, token);
}

export function createAchievement(
  token: string,
  input: { quarter: string; rawText: string; kpiId?: number | null },
): Promise<Achievement> {
  return apiFetch<Achievement>(
    "/achievements",
    {
      method: "POST",
      body: JSON.stringify(input),
    },
    token,
  );
}

export function updateAchievement(
  token: string,
  id: number,
  input: { quarter: string; rawText: string; enhancedText: string; category: string; impactNote: string; kpiId?: number | null },
): Promise<Achievement> {
  return apiFetch<Achievement>(
    `/achievements/${id}`,
    {
      method: "PUT",
      body: JSON.stringify(input),
    },
    token,
  );
}

export function deleteAchievement(token: string, id: number): Promise<void> {
  return apiFetch<void>(`/achievements/${id}`, { method: "DELETE" }, token);
}

export function enhanceAchievement(token: string, id: number): Promise<{ status: string }> {
  return apiFetch<{ status: string }>(`/achievements/${id}/enhance`, { method: "POST" }, token);
}

export function generateReport(token: string, quarter: string): Promise<Report> {
  return apiFetch<Report>(
    "/reports/generate",
    {
      method: "POST",
      body: JSON.stringify({ quarter }),
    },
    token,
  );
}

export function getReport(token: string, quarter: string): Promise<Report> {
  return apiFetch<Report>(`/reports/${encodeURIComponent(quarter)}`, undefined, token);
}
