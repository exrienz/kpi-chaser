export type KPI = {
  id: number;
  quarter: string;
  title: string;
  description: string;
  weight: number;
  targetMetric: string;
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

const API_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

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
