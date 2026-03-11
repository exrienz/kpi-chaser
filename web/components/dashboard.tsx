"use client";

import { FormEvent, useEffect, useState, useTransition } from "react";

import { Achievement, apiFetch, APIError, DashboardSummary, KPI, Report } from "../lib/api";
import { formatReport, ReportFormat } from "../lib/reportFormats";

type AuthResponse = {
  token: string;
  user: {
    email: string;
  };
};

const demoKPIs: KPI[] = [
  {
    id: 1,
    quarter: "2026-Q1",
    title: "Improve security posture",
    description: "Reduce attack surface and tighten control coverage.",
    weight: 40,
    targetMetric: "Complete 3 hardening initiatives",
  },
  {
    id: 2,
    quarter: "2026-Q1",
    title: "Automate response workflows",
    description: "Convert recurring manual tasks into scripts.",
    weight: 35,
    targetMetric: "Save 8 team hours per month",
  },
];

const demoAchievements: Achievement[] = [
  {
    id: 1,
    quarter: "2026-Q1",
    rawText: "Reviewed firewall rules and removed 14 unused policies",
    enhancedText:
      "Conducted a firewall policy review and removed 14 obsolete rules, reducing attack surface and improving baseline control hygiene.",
    category: "Process Improvement",
    impactNote: "Reduced unnecessary exposure across edge-facing services.",
    status: "enhanced",
  },
  {
    id: 2,
    quarter: "2026-Q1",
    rawText: "Built phishing alert triage script",
    enhancedText:
      "Developed an automated phishing triage script that streamlined repetitive alert review and increased analyst response capacity.",
    category: "Automation",
    impactNote: "Shifted a recurring manual task into a repeatable workflow.",
    status: "enhanced",
  },
];

const demoReport: Report = {
  id: 1,
  quarter: "2026-Q1",
  title: "Quarterly KPI Summary 2026-Q1",
  body:
    "Improve security posture\n- Conducted a firewall policy review and removed 14 obsolete rules, reducing attack surface and improving baseline control hygiene.\n\nAutomate response workflows\n- Developed an automated phishing triage script that streamlined repetitive alert review and increased analyst response capacity.",
};

const demoSummary: DashboardSummary = {
  quarter: "2026-Q1",
  totalKpis: 2,
  totalAchievements: 2,
  enhancedAchievements: 2,
  mappedAchievements: 1,
  draftAchievements: 0,
  pendingJobs: 0,
  kpiProgress: [
    {
      kpiId: 1,
      title: "Improve security posture",
      weight: 40,
      achievementCount: 1,
      enhancedCount: 1,
      progressPercent: 25,
    },
    {
      kpiId: 2,
      title: "Automate response workflows",
      weight: 35,
      achievementCount: 1,
      enhancedCount: 1,
      progressPercent: 25,
    },
  ],
};

export function Dashboard() {
  const [token, setToken] = useState<string>("");
  const [email, setEmail] = useState("owner@example.com");
  const [password, setPassword] = useState("changeme123");
  const [quarter, setQuarter] = useState("2026-Q1");
  const [rawText, setRawText] = useState("");
  const [selectedKpiId, setSelectedKpiId] = useState<string>("");
  const [kpiTitle, setKpiTitle] = useState("");
  const [kpiDescription, setKpiDescription] = useState("");
  const [kpiWeight, setKpiWeight] = useState("25");
  const [kpiTargetMetric, setKpiTargetMetric] = useState("");
  const [editingKpiId, setEditingKpiId] = useState<number | null>(null);
  const [editingAchievementId, setEditingAchievementId] = useState<number | null>(null);
  const [kpis, setKpis] = useState<KPI[]>(demoKPIs);
  const [achievements, setAchievements] = useState<Achievement[]>(demoAchievements);
  const [report, setReport] = useState<Report | null>(demoReport);
  const [summary, setSummary] = useState<DashboardSummary>(demoSummary);
  const [reportFormat, setReportFormat] = useState<ReportFormat>("review");
  const [error, setError] = useState<string>("");
  const [info, setInfo] = useState<string>("Demo data is shown until the API is connected.");
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    const saved = window.localStorage.getItem("kpi-journal-token");
    if (saved) {
      setToken(saved);
      hydrate(saved, quarter);
    }
  }, [quarter]);

  async function hydrate(currentToken: string, currentQuarter: string) {
    try {
      const [fetchedKPIs, fetchedAchievements, fetchedSummary] = await Promise.all([
        apiFetch<KPI[]>(`/kpis?quarter=${currentQuarter}`, undefined, currentToken),
        apiFetch<Achievement[]>(`/achievements?quarter=${currentQuarter}`, undefined, currentToken),
        apiFetch<DashboardSummary>(`/dashboard?quarter=${currentQuarter}`, undefined, currentToken),
      ]);
      setKpis(fetchedKPIs);
      setAchievements(fetchedAchievements);
      setSummary(fetchedSummary);
      try {
        const fetchedReport = await apiFetch<Report>(`/reports/${currentQuarter}`, undefined, currentToken);
        setReport(fetchedReport);
      } catch (err) {
        if (err instanceof APIError && err.status === 404) {
          setReport(null);
        } else {
          throw err;
        }
      }
      setInfo("Connected to backend API.");
      setError("");
    } catch (err) {
      setInfo("Backend unavailable. Showing seeded demo content.");
      setError(err instanceof Error ? err.message : "Unknown API error");
    }
  }

  async function handleAuth(event: FormEvent<HTMLFormElement>, mode: "login" | "register") {
    event.preventDefault();
    startTransition(async () => {
      try {
        const data = await apiFetch<AuthResponse>(`/auth/${mode}`, {
          method: "POST",
          body: JSON.stringify({ email, password }),
        });
        window.localStorage.setItem("kpi-journal-token", data.token);
        setToken(data.token);
        setInfo(`Authenticated as ${data.user.email}.`);
        setError("");
        await hydrate(data.token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Authentication failed");
      }
    });
  }

  async function handleAddKPI(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      setError("Log in or register before creating KPIs.");
      return;
    }
    startTransition(async () => {
      try {
        const created = await apiFetch<KPI>(
          "/kpis",
          {
            method: "POST",
            body: JSON.stringify({
              quarter,
              title: kpiTitle,
              description: kpiDescription,
              weight: Number(kpiWeight) || 0,
              targetMetric: kpiTargetMetric,
            }),
          },
          token,
        );
        setKpis((current) => (editingKpiId ? current.map((item) => (item.id === editingKpiId ? created : item)) : [created, ...current]));
        resetKpiForm();
        setError("");
        setEditingKpiId(null);
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not save KPI");
      }
    });
  }

  async function handleAddAchievement(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      setError("Log in or register before saving achievements.");
      return;
    }
    startTransition(async () => {
      try {
        const created = await apiFetch<Achievement>(
          "/achievements",
          {
            method: "POST",
            body: JSON.stringify({
              quarter,
              rawText,
              kpiId: selectedKpiId ? Number(selectedKpiId) : null,
            }),
          },
          token,
        );
        setAchievements((current) =>
          editingAchievementId ? current.map((item) => (item.id === editingAchievementId ? created : item)) : [created, ...current],
        );
        resetAchievementForm();
        setEditingAchievementId(null);
        setError("");
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not save achievement");
      }
    });
  }

  async function handleUpdateKPI(id: number) {
    if (!token) {
      setError("Log in or register before editing KPIs.");
      return;
    }
    startTransition(async () => {
      try {
        const updated = await apiFetch<KPI>(
          `/kpis/${id}`,
          {
            method: "PUT",
            body: JSON.stringify({
              quarter,
              title: kpiTitle,
              description: kpiDescription,
              weight: Number(kpiWeight) || 0,
              targetMetric: kpiTargetMetric,
            }),
          },
          token,
        );
        setKpis((current) => current.map((item) => (item.id === id ? updated : item)));
        resetKpiForm();
        setEditingKpiId(null);
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not update KPI");
      }
    });
  }

  async function handleUpdateAchievement(id: number, current: Achievement) {
    if (!token) {
      setError("Log in or register before editing achievements.");
      return;
    }
    startTransition(async () => {
      try {
        const updated = await apiFetch<Achievement>(
          `/achievements/${id}`,
          {
            method: "PUT",
            body: JSON.stringify({
              quarter,
              rawText: editingAchievementId === id ? rawText : current.rawText,
              enhancedText: current.enhancedText,
              category: current.category,
              impactNote: current.impactNote,
              kpiId: selectedKpiId ? Number(selectedKpiId) : null,
            }),
          },
          token,
        );
        setAchievements((items) => items.map((item) => (item.id === id ? updated : item)));
        resetAchievementForm();
        setEditingAchievementId(null);
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not update achievement");
      }
    });
  }

  async function handleEnhance(id: number) {
    if (!token) {
      setError("Log in or register before queuing AI enhancement.");
      return;
    }
    startTransition(async () => {
      try {
        await apiFetch<{ status: string }>(`/achievements/${id}/enhance`, { method: "POST" }, token);
        setInfo("Enhancement job queued. Start the Go worker to process it.");
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not queue enhancement");
      }
    });
  }

  async function handleGenerateReport() {
    if (!token) {
      setError("Log in or register before generating reports.");
      return;
    }
    startTransition(async () => {
      try {
        const generated = await apiFetch<Report>(
          "/reports/generate",
          {
            method: "POST",
            body: JSON.stringify({ quarter }),
          },
          token,
        );
        setReport(generated);
        setError("");
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not generate report");
      }
    });
  }

  async function handleDeleteKPI(id: number) {
    if (!token) {
      setError("Log in or register before deleting KPIs.");
      return;
    }
    startTransition(async () => {
      try {
        await apiFetch<void>(`/kpis/${id}`, { method: "DELETE" }, token);
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not delete KPI");
      }
    });
  }

  async function handleDeleteAchievement(id: number) {
    if (!token) {
      setError("Log in or register before deleting achievements.");
      return;
    }
    startTransition(async () => {
      try {
        await apiFetch<void>(`/achievements/${id}`, { method: "DELETE" }, token);
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not delete achievement");
      }
    });
  }

  async function handleCopyReport() {
    try {
      await navigator.clipboard.writeText(formatReport(report, reportFormat));
      setInfo(`Copied ${reportFormat} report format to clipboard.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not copy report");
    }
  }

  function handleDownloadReport() {
    const content = formatReport(report, reportFormat);
    const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${quarter}-${reportFormat}-report.txt`;
    link.click();
    URL.revokeObjectURL(url);
  }

  function startEditKpi(item: KPI) {
    setEditingKpiId(item.id);
    setKpiTitle(item.title);
    setKpiDescription(item.description);
    setKpiWeight(String(item.weight));
    setKpiTargetMetric(item.targetMetric);
  }

  function startEditAchievement(item: Achievement) {
    setEditingAchievementId(item.id);
    setRawText(item.rawText);
    setSelectedKpiId(item.kpiId ? String(item.kpiId) : "");
  }

  function resetKpiForm() {
    setKpiTitle("");
    setKpiDescription("");
    setKpiWeight("25");
    setKpiTargetMetric("");
  }

  function resetAchievementForm() {
    setRawText("");
    setSelectedKpiId("");
  }

  function kpiTitleFor(id?: number | null) {
    return kpis.find((item) => item.id === id)?.title ?? "Unmapped";
  }

  return (
    <main className="shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Self-hosted KPI Journal</p>
          <h1>Daily work logs turned into KPI-ready review narratives.</h1>
          <p className="lede">
            Capture small wins, enrich them with AI, and assemble a quarterly summary without reconstructing months of work from memory.
          </p>
        </div>
        <div className="statusCard">
          <span>Status</span>
          <strong>{info}</strong>
          <small>{isPending ? "Working..." : "Idle"}</small>
        </div>
      </section>

      <section className="grid two">
        <form className="panel" onSubmit={(event) => handleAuth(event, "login")}>
          <h2>Authentication</h2>
          <p>Email + password auth for private multi-user deployments.</p>
          <label>
            Email
            <input value={email} onChange={(event) => setEmail(event.target.value)} type="email" />
          </label>
          <label>
            Password
            <input value={password} onChange={(event) => setPassword(event.target.value)} type="password" />
          </label>
          <div className="row">
            <button type="submit">Login</button>
            <button
              type="button"
              className="ghost"
              onClick={() =>
                startTransition(async () => {
                  try {
                    const data = await apiFetch<AuthResponse>("/auth/register", {
                      method: "POST",
                      body: JSON.stringify({ email, password }),
                    });
                    window.localStorage.setItem("kpi-journal-token", data.token);
                    setToken(data.token);
                    setInfo(`Authenticated as ${data.user.email}.`);
                    setError("");
                    await hydrate(data.token, quarter);
                  } catch (err) {
                    setError(err instanceof Error ? err.message : "Registration failed");
                  }
                })
              }
            >
              Register
            </button>
          </div>
        </form>

        <div className="panel quarterPanel">
          <h2>Current Quarter</h2>
          <label>
            Quarter tag
            <input value={quarter} onChange={(event) => setQuarter(event.target.value)} />
          </label>
          <p>Examples: `2026-Q1`, `2026-Q2`.</p>
          {token ? <p className="chip">API token loaded</p> : <p className="chip muted">Using demo mode</p>}
          {error ? <p className="error">{error}</p> : null}
        </div>
      </section>

      <section className="grid two">
        <div className="panel">
          <div className="sectionHeader">
            <h2>Quarter Snapshot</h2>
            <span>{summary.quarter}</span>
          </div>
          <div className="statsGrid">
            <article className="statCard">
              <strong>{summary.totalKpis}</strong>
              <span>KPI targets</span>
            </article>
            <article className="statCard">
              <strong>{summary.totalAchievements}</strong>
              <span>Achievements logged</span>
            </article>
            <article className="statCard">
              <strong>{summary.enhancedAchievements}</strong>
              <span>AI enhanced</span>
            </article>
            <article className="statCard">
              <strong>{summary.mappedAchievements}</strong>
              <span>Mapped to KPI</span>
            </article>
          </div>
        </div>

        <div className="panel">
          <div className="sectionHeader">
            <h2>Queue Health</h2>
            <span>{summary.pendingJobs} pending</span>
          </div>
          <div className="statsGrid slim">
            <article className="statCard">
              <strong>{summary.draftAchievements}</strong>
              <span>Draft entries</span>
            </article>
            <article className="statCard">
              <strong>{summary.pendingJobs}</strong>
              <span>Open jobs</span>
            </article>
          </div>
        </div>
      </section>

      <section className="panel">
        <div className="sectionHeader">
          <h2>KPI Progress</h2>
          <span>{summary.kpiProgress.length} tracked</span>
        </div>
        <div className="stack">
          {summary.kpiProgress.map((item) => (
            <article key={item.kpiId} className="card">
              <div className="cardTop">
                <strong>{item.title}</strong>
                <span>{item.progressPercent}%</span>
              </div>
              <div className="progressBar">
                <span style={{ width: `${item.progressPercent}%` }} />
              </div>
              <small>
                {item.achievementCount} achievements, {item.enhancedCount} enhanced, weight {item.weight}%
              </small>
            </article>
          ))}
        </div>
      </section>

      <section className="grid two">
        <form
          className="panel"
          onSubmit={(event) => {
            event.preventDefault();
            if (editingKpiId) {
              void handleUpdateKPI(editingKpiId);
              return;
            }
            void handleAddKPI(event);
          }}
        >
          <h2>{editingKpiId ? "Edit KPI Target" : "Set KPI Targets"}</h2>
          <label>
            KPI title
            <input value={kpiTitle} onChange={(event) => setKpiTitle(event.target.value)} placeholder="Improve system security posture" />
          </label>
          <label>
            Description
            <textarea
              value={kpiDescription}
              onChange={(event) => setKpiDescription(event.target.value)}
              rows={4}
              placeholder="Describe the measurable outcome or target."
            />
          </label>
          <div className="gridForm">
            <label>
              Weight %
              <input value={kpiWeight} onChange={(event) => setKpiWeight(event.target.value)} type="number" min="0" max="100" />
            </label>
            <label>
              Target metric
              <input value={kpiTargetMetric} onChange={(event) => setKpiTargetMetric(event.target.value)} placeholder="Save 8 hours/month" />
            </label>
          </div>
          <div className="row">
            <button type="submit">{editingKpiId ? "Save KPI" : "Create KPI"}</button>
            {editingKpiId ? (
              <button
                type="button"
                className="ghost"
                onClick={() => {
                  setEditingKpiId(null);
                  resetKpiForm();
                }}
              >
                Cancel
              </button>
            ) : null}
          </div>
        </form>

        <form
          className="panel"
          onSubmit={(event) => {
            event.preventDefault();
            if (editingAchievementId) {
              const current = achievements.find((item) => item.id === editingAchievementId);
              if (current) {
                void handleUpdateAchievement(editingAchievementId, current);
              }
              return;
            }
            void handleAddAchievement(event);
          }}
        >
          <h2>{editingAchievementId ? "Edit Achievement" : "Log Daily Achievement"}</h2>
          <label>
            What did you accomplish?
            <textarea
              value={rawText}
              onChange={(event) => setRawText(event.target.value)}
              rows={5}
              placeholder="Reviewed firewall rules and removed unused policies"
            />
          </label>
          <label>
            KPI tag
            <select value={selectedKpiId} onChange={(event) => setSelectedKpiId(event.target.value)}>
              <option value="">Let AI map this later</option>
              {kpis.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.title}
                </option>
              ))}
            </select>
          </label>
          <div className="row">
            <button type="submit">{editingAchievementId ? "Save Changes" : "Save Achievement"}</button>
            {editingAchievementId ? (
              <button
                type="button"
                className="ghost"
                onClick={() => {
                  setEditingAchievementId(null);
                  resetAchievementForm();
                }}
              >
                Cancel
              </button>
            ) : null}
          </div>
        </form>
      </section>

      <section className="grid two">
        <div className="panel">
          <div className="sectionHeader">
            <h2>KPI Targets</h2>
            <span>{kpis.length} total</span>
          </div>
          <div className="stack">
            {kpis.map((item) => (
              <article key={item.id} className="card">
                <div className="cardTop">
                  <strong>{item.title}</strong>
                  <span>{item.weight}%</span>
                </div>
                <p>{item.description}</p>
                <small>{item.targetMetric}</small>
                <div className="row">
                  <button type="button" className="ghost" onClick={() => startEditKpi(item)}>
                    Edit KPI
                  </button>
                  <button type="button" className="ghost danger" onClick={() => void handleDeleteKPI(item.id)}>
                    Delete
                  </button>
                </div>
              </article>
            ))}
          </div>
        </div>

        <div className="panel">
          <div className="sectionHeader">
            <h2>Achievements</h2>
            <span>{achievements.length} logged</span>
          </div>
          <div className="stack">
            {achievements.map((item) => (
              <article key={item.id} className="card">
                <div className="cardTop">
                  <strong>{item.category || "Draft entry"}</strong>
                  <span>{item.status}</span>
                </div>
                <p>{item.enhancedText || item.rawText}</p>
                <small>{item.impactNote || "Impact note pending AI enrichment."}</small>
                <small>Mapped KPI: {kpiTitleFor(item.kpiId)}</small>
                <div className="row">
                  <button type="button" className="ghost" onClick={() => void handleEnhance(item.id)}>
                    Enhance with AI
                  </button>
                  <button type="button" className="ghost" onClick={() => startEditAchievement(item)}>
                    Edit / Retag
                  </button>
                  <button type="button" className="ghost danger" onClick={() => void handleDeleteAchievement(item.id)}>
                    Delete
                  </button>
                </div>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="panel reportPanel">
        <div className="sectionHeader">
          <div>
            <h2>Quarterly Report</h2>
            <p>Generate structured quarterly output from saved achievements.</p>
          </div>
          <div className="row">
            <button type="button" className="ghost" onClick={() => void handleCopyReport()}>
              Copy
            </button>
            <button type="button" className="ghost" onClick={handleDownloadReport}>
              Download
            </button>
            <button type="button" onClick={() => void handleGenerateReport()}>
              Generate KPI Summary
            </button>
          </div>
        </div>
        <div className="formatTabs">
          {(["review", "notion", "hr"] as ReportFormat[]).map((item) => (
            <button
              key={item}
              type="button"
              className={reportFormat === item ? "tab active" : "tab"}
              onClick={() => setReportFormat(item)}
            >
              {item}
            </button>
          ))}
        </div>
        <div className="reportBody">{formatReport(report, reportFormat)}</div>
      </section>
    </main>
  );
}
