"use client";

import { CSSProperties, FormEvent, ReactNode, useEffect, useMemo, useState, useTransition } from "react";

import {
  Achievement,
  APIError,
  DashboardSummary,
  KPI,
  Report,
  apiFetch,
  createAchievement,
  createKPI,
  createSubKPI,
  deleteAchievement,
  deleteKPI,
  enhanceAchievement,
  generateReport,
  getDashboardSummary,
  getReport,
  listAchievements,
  listKPIHierarchy,
  listKPIs,
  updateAchievement,
  updateKPI,
  updateKPIProgress,
} from "../lib/api";
import { formatReport, ReportFormat } from "../lib/reportFormats";

type AuthResponse = {
  token: string;
  user: {
    email: string;
  };
};

type QuarterKey = "progressQ1" | "progressQ2" | "progressQ3" | "progressQ4";

type ProgressDraft = {
  progressQ1: number;
  progressQ2: number;
  progressQ3: number;
  progressQ4: number;
};

const demoKPIs: KPI[] = [
  {
    id: 1,
    quarter: "2026-Q1",
    title: "Improve security posture",
    description: "Reduce attack surface and tighten control coverage.",
    weight: 40,
    targetMetric: "Complete 3 hardening initiatives",
    parentKpiId: null,
    progressQ1: 55,
    progressQ2: 50,
    progressQ3: 45,
    progressQ4: 60,
    annualProgress: 52,
  },
  {
    id: 2,
    quarter: "2026-Q1",
    title: "Automate response workflows",
    description: "Convert recurring manual tasks into scripts.",
    weight: 35,
    targetMetric: "Save 8 team hours per month",
    parentKpiId: null,
    progressQ1: 35,
    progressQ2: 40,
    progressQ3: 45,
    progressQ4: 50,
    annualProgress: 42,
  },
  {
    id: 3,
    quarter: "2026-Q1",
    title: "Firewall hardening",
    description: "Improve inbound and outbound rule hygiene.",
    weight: 20,
    targetMetric: "Remove obsolete policies",
    parentKpiId: 1,
    progressQ1: 65,
    progressQ2: 60,
    progressQ3: 55,
    progressQ4: 60,
    annualProgress: 60,
  },
];

const demoHierarchy: KPI[] = [
  {
    ...demoKPIs[0],
    children: [{ ...demoKPIs[2] }],
  },
  {
    ...demoKPIs[1],
    children: [],
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
    kpiId: 3,
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
    kpiId: 2,
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
  totalKpis: 3,
  totalAchievements: 2,
  enhancedAchievements: 2,
  mappedAchievements: 2,
  draftAchievements: 0,
  pendingJobs: 0,
  kpiProgress: [
    {
      kpiId: 1,
      title: "Improve security posture",
      weight: 40,
      achievementCount: 1,
      enhancedCount: 1,
      progressPercent: 52,
    },
    {
      kpiId: 2,
      title: "Automate response workflows",
      weight: 35,
      achievementCount: 1,
      enhancedCount: 1,
      progressPercent: 42,
    },
  ],
};

function flattenKPIs(nodes: KPI[]): KPI[] {
  const output: KPI[] = [];
  const walk = (items: KPI[]) => {
    for (const item of items) {
      output.push(item);
      if (item.children && item.children.length > 0) {
        walk(item.children);
      }
    }
  };
  walk(nodes);
  return output;
}

function progressFromKPI(item: KPI): ProgressDraft {
  return {
    progressQ1: item.progressQ1,
    progressQ2: item.progressQ2,
    progressQ3: item.progressQ3,
    progressQ4: item.progressQ4,
  };
}

function normalizeProgress(value: number): number {
  if (Number.isNaN(value)) {
    return 0;
  }
  if (value < 0) {
    return 0;
  }
  if (value > 100) {
    return 100;
  }
  return value;
}

export function Dashboard() {
  const [token, setToken] = useState<string>("");
  const [email, setEmail] = useState("owner@example.com");
  const [password, setPassword] = useState("changeme123");
  const [quarter, setQuarter] = useState("2026-Q1");
  const [rawText, setRawText] = useState("");
  const [selectedKpiId, setSelectedKpiId] = useState<string>("");
  const [kpiParentId, setKpiParentId] = useState<string>("");
  const [kpiTitle, setKpiTitle] = useState("");
  const [kpiDescription, setKpiDescription] = useState("");
  const [kpiWeight, setKpiWeight] = useState("25");
  const [kpiTargetMetric, setKpiTargetMetric] = useState("");
  const [editingKpiId, setEditingKpiId] = useState<number | null>(null);
  const [editingAchievementId, setEditingAchievementId] = useState<number | null>(null);
  const [kpis, setKpis] = useState<KPI[]>(demoKPIs);
  const [kpiHierarchy, setKpiHierarchy] = useState<KPI[]>(demoHierarchy);
  const [achievements, setAchievements] = useState<Achievement[]>(demoAchievements);
  const [report, setReport] = useState<Report | null>(demoReport);
  const [summary, setSummary] = useState<DashboardSummary>(demoSummary);
  const [reportFormat, setReportFormat] = useState<ReportFormat>("review");
  const [progressDrafts, setProgressDrafts] = useState<Record<number, ProgressDraft>>({});
  const [expandedNodes, setExpandedNodes] = useState<Record<number, boolean>>({});
  const [error, setError] = useState<string>("");
  const [info, setInfo] = useState<string>("Demo data is shown until the API is connected.");
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    const saved = window.localStorage.getItem("kpi-journal-token");
    if (saved) {
      setToken(saved);
      void hydrate(saved, quarter);
    }
  }, [quarter]);

  const allKpis = useMemo(() => flattenKPIs(kpiHierarchy.length > 0 ? kpiHierarchy : kpis), [kpiHierarchy, kpis]);

  const kpiLookup = useMemo(() => {
    const map = new Map<number, KPI>();
    for (const item of allKpis) {
      map.set(item.id, item);
    }
    return map;
  }, [allKpis]);

  async function hydrate(currentToken: string, currentQuarter: string) {
    try {
      const [fetchedKPIs, fetchedHierarchy, fetchedAchievements, fetchedSummary] = await Promise.all([
        listKPIs(currentToken, currentQuarter),
        listKPIHierarchy(currentToken, currentQuarter),
        listAchievements(currentToken, currentQuarter),
        getDashboardSummary(currentToken, currentQuarter),
      ]);
      setKpis(fetchedKPIs ?? []);
      setKpiHierarchy(fetchedHierarchy ?? []);
      setAchievements(fetchedAchievements ?? []);
      setSummary(fetchedSummary ?? demoSummary);
      setProgressDrafts({});

      try {
        const fetchedReport = await getReport(currentToken, currentQuarter);
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
      setKpis(demoKPIs);
      setKpiHierarchy(demoHierarchy);
      setAchievements(demoAchievements);
      setSummary(demoSummary);
      setReport(demoReport);
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

  async function handleRegister() {
    if (!email || !password) {
      setError("Email and password are required.");
      return;
    }
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
    });
  }

  async function handleAddKPI() {
    if (!token) {
      setError("Log in or register before creating KPIs.");
      return;
    }

    const weight = Number(kpiWeight) || 0;
    const parentID = kpiParentId ? Number(kpiParentId) : null;

    startTransition(async () => {
      try {
        if (parentID) {
          await createSubKPI(token, parentID, {
            title: kpiTitle,
            description: kpiDescription,
            weight,
            targetMetric: kpiTargetMetric,
          });
        } else {
          await createKPI(token, {
            quarter,
            title: kpiTitle,
            description: kpiDescription,
            weight,
            targetMetric: kpiTargetMetric,
            parentKpiId: null,
          });
        }

        resetKpiForm();
        setEditingKpiId(null);
        setError("");
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not save KPI");
      }
    });
  }

  async function handleUpdateKPI(id: number) {
    if (!token) {
      setError("Log in or register before editing KPIs.");
      return;
    }

    const current = kpiLookup.get(id);
    if (!current) {
      setError("Unable to locate KPI for update.");
      return;
    }

    startTransition(async () => {
      try {
        await updateKPI(token, id, {
          quarter,
          title: kpiTitle,
          description: kpiDescription,
          weight: Number(kpiWeight) || 0,
          targetMetric: kpiTargetMetric,
          parentKpiId: kpiParentId ? Number(kpiParentId) : null,
          progressQ1: current.progressQ1,
          progressQ2: current.progressQ2,
          progressQ3: current.progressQ3,
          progressQ4: current.progressQ4,
        });
        resetKpiForm();
        setEditingKpiId(null);
        setError("");
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not update KPI");
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
        await deleteKPI(token, id);
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not delete KPI");
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
        await createAchievement(token, {
          quarter,
          rawText,
          kpiId: selectedKpiId ? Number(selectedKpiId) : null,
        });
        resetAchievementForm();
        setEditingAchievementId(null);
        setError("");
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not save achievement");
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
        await updateAchievement(token, id, {
          quarter,
          rawText: editingAchievementId === id ? rawText : current.rawText,
          enhancedText: current.enhancedText,
          category: current.category,
          impactNote: current.impactNote,
          kpiId: selectedKpiId ? Number(selectedKpiId) : null,
        });
        setEditingAchievementId(null);
        resetAchievementForm();
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
        await enhanceAchievement(token, id);
        setInfo("Enhancement job queued. Start the Go worker to process it.");
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not queue enhancement");
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
        await deleteAchievement(token, id);
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not delete achievement");
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
        const generated = await generateReport(token, quarter);
        setReport(generated);
        setError("");
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not generate report");
      }
    });
  }

  async function handleSaveProgress(id: number) {
    if (!token) {
      setError("Log in or register before updating progress.");
      return;
    }

    const current = kpiLookup.get(id);
    if (!current) {
      setError("Unable to locate KPI for progress update.");
      return;
    }

    const draft = progressDrafts[id] ?? progressFromKPI(current);

    startTransition(async () => {
      try {
        await updateKPIProgress(token, id, draft);
        setProgressDrafts((state) => {
          const next = { ...state };
          delete next[id];
          return next;
        });
        await hydrate(token, quarter);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not update progress");
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

  function updateProgressDraft(id: number, key: QuarterKey, value: string) {
    const nextValue = normalizeProgress(Number(value));
    setProgressDrafts((current) => {
      const kpi = kpiLookup.get(id);
      const base =
        current[id] ??
        (kpi
          ? progressFromKPI(kpi)
          : {
              progressQ1: 0,
              progressQ2: 0,
              progressQ3: 0,
              progressQ4: 0,
            });
      return {
        ...current,
        [id]: {
          ...base,
          [key]: nextValue,
        },
      };
    });
  }

  function toggleNode(id: number) {
    setExpandedNodes((state) => ({
      ...state,
      [id]: !(state[id] ?? true),
    }));
  }

  function startEditKpi(item: KPI) {
    setEditingKpiId(item.id);
    setKpiTitle(item.title);
    setKpiDescription(item.description);
    setKpiWeight(String(item.weight));
    setKpiTargetMetric(item.targetMetric);
    setKpiParentId(item.parentKpiId ? String(item.parentKpiId) : "");
  }

  function startSubKpi(parent: KPI) {
    resetKpiForm();
    setEditingKpiId(null);
    setKpiParentId(String(parent.id));
    setInfo(`Creating sub-KPI under \"${parent.title}\".`);
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
    setKpiParentId("");
  }

  function resetAchievementForm() {
    setRawText("");
    setSelectedKpiId("");
  }

  function kpiTitleFor(id?: number | null) {
    if (!id) {
      return "Unmapped";
    }
    return kpiLookup.get(id)?.title ?? "Unknown KPI";
  }

  function renderHierarchy(nodes: KPI[], depth = 0): ReactNode {
    return nodes.map((item) => {
      const hasChildren = (item.children?.length ?? 0) > 0;
      const isExpanded = expandedNodes[item.id] ?? true;
      const draft = progressDrafts[item.id] ?? progressFromKPI(item);
      const depthStyle = { "--depth": depth } as CSSProperties;

      return (
        <div key={item.id} className="treeNode" style={depthStyle}>
          <article className="card hierarchyCard">
            <div className="cardTop">
              <div className="kpiNodeTitle">
                {hasChildren ? (
                  <button type="button" className="treeToggle" onClick={() => toggleNode(item.id)}>
                    {isExpanded ? "-" : "+"}
                  </button>
                ) : (
                  <span className="treeDot" />
                )}
                <strong>{item.title}</strong>
                {item.parentKpiId ? <span className="kpiBadge child">Sub-KPI</span> : null}
                {hasChildren ? <span className="kpiBadge parent">Parent</span> : null}
              </div>
              <span>{item.annualProgress}% annual</span>
            </div>
            <p>{item.description}</p>
            <small>Target: {item.targetMetric || "Not set"}</small>
            <small>
              Weight {item.weight}% • Quarter {item.quarter}
            </small>

            <div className="quarterGrid">
              {([
                ["Q1", "progressQ1"],
                ["Q2", "progressQ2"],
                ["Q3", "progressQ3"],
                ["Q4", "progressQ4"],
              ] as [string, QuarterKey][]).map(([label, key]) => (
                <label key={label} className="quarterCell">
                  <span>{label}</span>
                  <input
                    type="range"
                    min="0"
                    max="100"
                    value={draft[key]}
                    onChange={(event) => updateProgressDraft(item.id, key, event.target.value)}
                  />
                  <input
                    type="number"
                    min="0"
                    max="100"
                    value={draft[key]}
                    onChange={(event) => updateProgressDraft(item.id, key, event.target.value)}
                  />
                </label>
              ))}
            </div>

            <div className="row">
              <button type="button" className="ghost" onClick={() => void handleSaveProgress(item.id)}>
                Save Progress
              </button>
              <button type="button" className="ghost" onClick={() => startEditKpi(item)}>
                Edit
              </button>
              <button type="button" className="ghost" onClick={() => startSubKpi(item)}>
                Add Sub-KPI
              </button>
              <button type="button" className="ghost danger" onClick={() => void handleDeleteKPI(item.id)}>
                Delete
              </button>
            </div>
          </article>

          {hasChildren && isExpanded ? <div className="treeChildren">{renderHierarchy(item.children ?? [], depth + 1)}</div> : null}
        </div>
      );
    });
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
        <form className="panel" onSubmit={(event) => void handleAuth(event, "login")}>
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
            <button type="button" className="ghost" onClick={() => void handleRegister()}>
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
          <h2>KPI Hierarchy & Progress</h2>
          <span>{allKpis.length} nodes</span>
        </div>
        <p>Expand parent KPIs, manage sub-KPIs, and update Q1-Q4 progress in place.</p>
        <div className="stack treeWrap">{kpiHierarchy.length > 0 ? renderHierarchy(kpiHierarchy) : <p>No KPI targets yet.</p>}</div>
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
            void handleAddKPI();
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
          <label>
            Parent KPI
            <select value={kpiParentId} onChange={(event) => setKpiParentId(event.target.value)}>
              <option value="">Root KPI (no parent)</option>
              {allKpis
                .filter((item) => item.id !== editingKpiId)
                .map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.title}
                  </option>
                ))}
            </select>
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
            <button type="submit">{editingKpiId ? "Save KPI" : kpiParentId ? "Create Sub-KPI" : "Create KPI"}</button>
            {editingKpiId || kpiParentId ? (
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
              {allKpis.map((item) => (
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
            <h2>KPI Progress</h2>
            <span>{(summary.kpiProgress ?? []).length} tracked</span>
          </div>
          <div className="stack">
            {(summary.kpiProgress ?? []).map((item) => (
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
            <button key={item} type="button" className={reportFormat === item ? "tab active" : "tab"} onClick={() => setReportFormat(item)}>
              {item}
            </button>
          ))}
        </div>
        <div className="reportBody">{formatReport(report, reportFormat)}</div>
      </section>
    </main>
  );
}
