"use client";

import { FormEvent, ReactNode, useEffect, useMemo, useState, useTransition } from "react";

import type { Achievement, AuthUser, KPI, Report } from "../lib/api";
import {
  APIError,
  createAchievement,
  createKPI,
  createSubKPI,
  deleteKPI,
  generateReport,
  getMe,
  getReport,
  listAchievements,
  listKPIHierarchy,
  login,
  register,
  resetAllProgress,
} from "../lib/api";
import { groupAchievementsByKpi, groupKpisByQuarter, quarterProgressForKpi, unmappedAchievements } from "../lib/dashboardData";

type AuthMode = "login" | "register";

type KpiFormState = {
  quarter: string;
  title: string;
  description: string;
  weight: string;
  targetMetric: string;
  parentId: string;
};

type AchievementFormState = {
  quarter: string;
  rawText: string;
  kpiId: string;
};

type ConfirmState =
  | {
      title: string;
      message: string;
      action: { type: "delete-kpi"; id: number };
    }
  | null;

const AUTH_STORAGE_KEY = "kpi-journal-token";

const initialKpiForm: KpiFormState = {
  quarter: "2026-Q1",
  title: "",
  description: "",
  weight: "25",
  targetMetric: "",
  parentId: "",
};

const initialAchievementForm: AchievementFormState = {
  quarter: "2026-Q1",
  rawText: "",
  kpiId: "",
};

export function DashboardRedesign() {
  const [token, setToken] = useState("");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [authMode, setAuthMode] = useState<AuthMode>("login");
  const [email, setEmail] = useState("owner@example.com");
  const [password, setPassword] = useState("changeme123");
  const [kpiForm, setKpiForm] = useState<KpiFormState>(initialKpiForm);
  const [achievementForm, setAchievementForm] = useState<AchievementFormState>(initialAchievementForm);
  const [kpiHierarchy, setKpiHierarchy] = useState<KPI[]>([]);
  const [achievements, setAchievements] = useState<Achievement[]>([]);
  const [reports, setReports] = useState<Record<string, Report | null | undefined>>({});
  const [selectedYear, setSelectedYear] = useState("2026");
  const [selectedReportQuarter, setSelectedReportQuarter] = useState("2026-Q1");
  const [confirmState, setConfirmState] = useState<ConfirmState>(null);
  const [resetOpen, setResetOpen] = useState(false);
  const [resetPassword, setResetPassword] = useState("");
  const [resetConfirmation, setResetConfirmation] = useState("");
  const [statusMessage, setStatusMessage] = useState("Authentication is required before dashboard data is shown.");
  const [error, setError] = useState("");
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    const savedToken = window.localStorage.getItem(AUTH_STORAGE_KEY);
    if (!savedToken) {
      return;
    }
    startTransition(async () => {
      try {
        const currentUser = await getMe(savedToken);
        setToken(savedToken);
        setUser(currentUser);
        await hydrate(savedToken);
      } catch {
        window.localStorage.removeItem(AUTH_STORAGE_KEY);
      }
    });
  }, []);

  const quarterGroups = useMemo(() => groupKpisByQuarter(kpiHierarchy), [kpiHierarchy]);
  const years = useMemo(() => [...new Set(quarterGroups.map((group) => group.year))].sort(), [quarterGroups]);
  const visibleQuarterGroups = useMemo(
    () => quarterGroups.filter((group) => group.year === selectedYear),
    [quarterGroups, selectedYear],
  );
  const flatKpis = useMemo(() => flatten(kpiHierarchy), [kpiHierarchy]);
  const quarterLookup = useMemo(() => new Set(quarterGroups.map((group) => group.quarter)), [quarterGroups]);
  const kpiOptions = useMemo(
    () => flatKpis.map((item) => ({ id: item.id, title: item.title, quarter: item.quarter })),
    [flatKpis],
  );

  useEffect(() => {
    if (years.length === 0) {
      return;
    }
    if (!years.includes(selectedYear)) {
      setSelectedYear(years[0]);
    }
  }, [selectedYear, years]);

  useEffect(() => {
    if (visibleQuarterGroups.length === 0) {
      return;
    }
    const firstQuarter = visibleQuarterGroups[0]?.quarter ?? "";
    if (!quarterLookup.has(selectedReportQuarter) || !selectedReportQuarter.startsWith(selectedYear)) {
      setSelectedReportQuarter(firstQuarter);
    }
    setKpiForm((current) => ({
      ...current,
      quarter: current.parentId ? current.quarter : firstQuarter,
    }));
    setAchievementForm((current) => ({
      ...current,
      quarter: firstQuarter,
    }));
  }, [quarterLookup, selectedReportQuarter, selectedYear, visibleQuarterGroups]);

  useEffect(() => {
    if (!token || !selectedReportQuarter) {
      return;
    }
    if (reports[selectedReportQuarter] !== undefined) {
      return;
    }
    void loadReport(token, selectedReportQuarter);
  }, [reports, selectedReportQuarter, token]);

  async function hydrate(currentToken: string) {
    const [nextHierarchy, nextAchievements] = await Promise.all([
      listKPIHierarchy(currentToken, ""),
      listAchievements(currentToken, ""),
    ]);
    setKpiHierarchy(nextHierarchy);
    setAchievements(nextAchievements);
    setError("");
    setStatusMessage("Dashboard synced.");
  }

  async function loadReport(currentToken: string, quarter: string) {
    try {
      const report = await getReport(currentToken, quarter);
      setReports((current) => ({ ...current, [quarter]: report }));
    } catch (err) {
      if (err instanceof APIError && err.status === 404) {
        setReports((current) => ({ ...current, [quarter]: null }));
        return;
      }
      throw err;
    }
  }

  async function handleAuthenticate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    startTransition(async () => {
      try {
        const payload = { email, password };
        const response = authMode === "login" ? await login(payload) : await register(payload);
        window.localStorage.setItem(AUTH_STORAGE_KEY, response.token);
        setToken(response.token);
        setUser(response.user);
        setStatusMessage(`${authMode === "login" ? "Logged in" : "Registered"} as ${response.user.email}.`);
        await hydrate(response.token);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Authentication failed");
      }
    });
  }

  function handleLogout() {
    window.localStorage.removeItem(AUTH_STORAGE_KEY);
    setToken("");
    setUser(null);
    setKpiHierarchy([]);
    setAchievements([]);
    setReports({});
    setStatusMessage("Signed out.");
    setError("");
  }

  async function handleCreateKpi(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      return;
    }
    startTransition(async () => {
      try {
        if (kpiForm.parentId) {
          await createSubKPI(token, Number(kpiForm.parentId), {
            title: kpiForm.title,
            description: kpiForm.description,
            weight: Number(kpiForm.weight) || 0,
            targetMetric: kpiForm.targetMetric,
          });
        } else {
          await createKPI(token, {
            quarter: kpiForm.quarter,
            title: kpiForm.title,
            description: kpiForm.description,
            weight: Number(kpiForm.weight) || 0,
            targetMetric: kpiForm.targetMetric,
            parentKpiId: null,
          });
        }
        setKpiForm((current) => ({
          ...initialKpiForm,
          quarter: current.quarter,
        }));
        setStatusMessage("KPI saved.");
        await hydrate(token);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not save KPI");
      }
    });
  }

  async function handleCreateAchievement(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      return;
    }
    startTransition(async () => {
      try {
        await createAchievement(token, {
          quarter: achievementForm.quarter,
          rawText: achievementForm.rawText,
          kpiId: achievementForm.kpiId ? Number(achievementForm.kpiId) : null,
        });
        setAchievementForm((current) => ({
          ...initialAchievementForm,
          quarter: current.quarter,
        }));
        setStatusMessage("Achievement logged.");
        await hydrate(token);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not save achievement");
      }
    });
  }

  async function handleGenerateQuarterReport(quarter: string) {
    if (!token) {
      return;
    }
    startTransition(async () => {
      try {
        const report = await generateReport(token, quarter);
        setReports((current) => ({ ...current, [quarter]: report }));
        setSelectedReportQuarter(quarter);
        setStatusMessage(`${quarter} report refreshed.`);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not generate report");
      }
    });
  }

  async function handleConfirmAction() {
    if (!token || !confirmState) {
      return;
    }
    startTransition(async () => {
      try {
        if (confirmState.action.type === "delete-kpi") {
          await deleteKPI(token, confirmState.action.id);
          setStatusMessage("KPI deleted.");
          setConfirmState(null);
          await hydrate(token);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not complete action");
      }
    });
  }

  async function handleResetProgress(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      return;
    }
    startTransition(async () => {
      try {
        const result = await resetAllProgress(token, {
          password: resetPassword,
          confirmation: "RESET",
        });
        setResetOpen(false);
        setResetPassword("");
        setResetConfirmation("");
        setReports({});
        setStatusMessage(
          `Progress reset: ${result.kpisUpdated} KPIs reset, ${result.achievementsDeleted} achievements removed.`,
        );
        await hydrate(token);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not reset progress");
      }
    });
  }

  if (!user) {
    return (
      <main className="authShell">
        <section className="authPanel">
          <p className="eyebrow">Secure Access</p>
          <h1>Quarterly KPI tracking starts behind an auth gate.</h1>
          <p className="lede">
            Login or register before viewing dashboard metrics, quarterly reports, or reset controls.
          </p>
          <div className="authTabs">
            <button className={authMode === "login" ? "tab active" : "tab"} onClick={() => setAuthMode("login")} type="button">
              Login
            </button>
            <button className={authMode === "register" ? "tab active" : "tab"} onClick={() => setAuthMode("register")} type="button">
              Register
            </button>
          </div>
          <form className="authForm" onSubmit={handleAuthenticate}>
            <label>
              Email
              <input autoComplete="email" onChange={(event) => setEmail(event.target.value)} value={email} />
            </label>
            <label>
              Password
              <input
                autoComplete={authMode === "login" ? "current-password" : "new-password"}
                onChange={(event) => setPassword(event.target.value)}
                type="password"
                value={password}
              />
            </label>
            <button disabled={isPending} type="submit">
              {isPending ? "Working..." : authMode === "login" ? "Login to dashboard" : "Create account"}
            </button>
          </form>
          <p className="statusLine">{statusMessage}</p>
          {error ? <p className="errorBanner">{error}</p> : null}
        </section>
      </main>
    );
  }

  const report = reports[selectedReportQuarter] ?? null;
  const reportAchievements = groupAchievementsByKpi(achievements, selectedReportQuarter);
  const orphanAchievements = unmappedAchievements(achievements, selectedReportQuarter);
  const totalProgress = flatKpis.length
    ? Math.round(flatKpis.reduce((sum, item) => sum + quarterProgressForKpi(item), 0) / flatKpis.length)
    : 0;

  return (
    <main className="dashboardV2">
      <section className="topBar panel">
        <div>
          <p className="eyebrow">New Dashboard</p>
          <h2>Quarter, KPI, and sub-KPI progress with secure controls.</h2>
        </div>
        <div className="topBarActions">
          <select onChange={(event) => setSelectedYear(event.target.value)} value={selectedYear}>
            {years.map((year) => (
              <option key={year} value={year}>
                {year}
              </option>
            ))}
          </select>
          <span className="pill">{user.email}</span>
          <button className="ghost" onClick={handleLogout} type="button">
            Logout
          </button>
        </div>
      </section>

      <section className="overviewGrid">
        <article className="metricPanel panel">
          <span>Visible quarters</span>
          <strong>{visibleQuarterGroups.length}</strong>
        </article>
        <article className="metricPanel panel">
          <span>Tracked KPIs</span>
          <strong>{flatKpis.length}</strong>
        </article>
        <article className="metricPanel panel">
          <span>Logged achievements</span>
          <strong>{achievements.length}</strong>
        </article>
        <article className="metricPanel panel">
          <span>Average progress</span>
          <strong>{totalProgress}%</strong>
        </article>
      </section>

      <section className="workspaceGrid">
        <div className="mainColumn">
          <article className="panel">
            <div className="sectionHeader">
              <div>
                <p className="eyebrow">Quarter View</p>
                <h2>{selectedYear} KPI hierarchy</h2>
              </div>
              <span className="statusLine">{statusMessage}</span>
            </div>
            {visibleQuarterGroups.length === 0 ? (
              <p className="emptyState">No KPIs are available for this year yet.</p>
            ) : (
              <div className="quarterStack">
                {visibleQuarterGroups.map((group) => (
                  <section className="quarterCard" key={group.quarter}>
                    <div className="row">
                      <div>
                        <h3>{group.quarter}</h3>
                        <p className="mutedText">KPIs are nested under their quarter with progress percentages.</p>
                      </div>
                      <button className="ghost" onClick={() => handleGenerateQuarterReport(group.quarter)} type="button">
                        Generate report
                      </button>
                    </div>
                    <div className="treeList">
                      {group.items.map((item) => (
                        <KpiTree
                          achievementsByKpi={reportAchievements}
                          key={item.id}
                          onDelete={(id) =>
                            setConfirmState({
                              title: "Delete KPI",
                              message: "This removes the KPI and any nested sub-KPIs. Confirmation is required.",
                              action: { type: "delete-kpi", id },
                            })
                          }
                          onPrepareSubKpi={(parent) =>
                            setKpiForm((current) => ({
                              ...current,
                              quarter: parent.quarter,
                              parentId: String(parent.id),
                            }))
                          }
                          quarter={group.quarter}
                          reportQuarter={selectedReportQuarter}
                          root={item}
                        />
                      ))}
                    </div>
                  </section>
                ))}
              </div>
            )}
          </article>

          <article className="panel">
            <div className="sectionHeader">
              <div>
                <p className="eyebrow">Yearly Checkpoint</p>
                <h2>Quarterly reports aggregated by year</h2>
              </div>
            </div>
            <div className="reportTabs">
              {visibleQuarterGroups.map((group) => (
                <button
                  className={selectedReportQuarter === group.quarter ? "tab active" : "tab"}
                  key={group.quarter}
                  onClick={() => setSelectedReportQuarter(group.quarter)}
                  type="button"
                >
                  {group.quarter}
                </button>
              ))}
            </div>
            <div className="yearlyCheckpoint">
              <div className="yearlySummary">
                <strong>{selectedReportQuarter}</strong>
                <p>{report ? report.title : "Generate the quarterly report to persist a snapshot."}</p>
              </div>
              <div className="reportHierarchy">
                {visibleQuarterGroups
                  .filter((group) => group.quarter === selectedReportQuarter)
                  .flatMap((group) => group.items)
                  .map((item) => (
                    <ReportTree key={item.id} node={item} reportAchievements={reportAchievements} />
                  ))}
                {orphanAchievements.length > 0 ? (
                  <section className="reportBlock">
                    <h4>Unmapped achievements</h4>
                    <ul>
                      {orphanAchievements.map((item) => (
                        <li key={item.id}>{item.enhancedText || item.rawText}</li>
                      ))}
                    </ul>
                  </section>
                ) : null}
              </div>
            </div>
          </article>
        </div>

        <aside className="sideColumn">
          <article className="panel">
            <p className="eyebrow">Add KPI</p>
            <form className="stackForm" onSubmit={handleCreateKpi}>
              <label>
                Quarter
                <select
                  disabled={Boolean(kpiForm.parentId)}
                  onChange={(event) => setKpiForm((current) => ({ ...current, quarter: event.target.value }))}
                  value={kpiForm.quarter}
                >
                  {visibleQuarterGroups.map((group) => (
                    <option key={group.quarter} value={group.quarter}>
                      {group.quarter}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Parent KPI
                <select
                  onChange={(event) => setKpiForm((current) => ({ ...current, parentId: event.target.value }))}
                  value={kpiForm.parentId}
                >
                  <option value="">Root KPI</option>
                  {kpiOptions
                    .filter((option) => option.quarter === kpiForm.quarter)
                    .map((option) => (
                      <option key={option.id} value={option.id}>
                        {option.title}
                      </option>
                    ))}
                </select>
              </label>
              <label>
                Title
                <input onChange={(event) => setKpiForm((current) => ({ ...current, title: event.target.value }))} value={kpiForm.title} />
              </label>
              <label>
                Description
                <textarea
                  onChange={(event) => setKpiForm((current) => ({ ...current, description: event.target.value }))}
                  rows={3}
                  value={kpiForm.description}
                />
              </label>
              <label>
                Weight
                <input onChange={(event) => setKpiForm((current) => ({ ...current, weight: event.target.value }))} value={kpiForm.weight} />
              </label>
              <label>
                Target metric
                <input
                  onChange={(event) => setKpiForm((current) => ({ ...current, targetMetric: event.target.value }))}
                  value={kpiForm.targetMetric}
                />
              </label>
              <button disabled={isPending} type="submit">
                Save KPI
              </button>
            </form>
          </article>

          <article className="panel">
            <p className="eyebrow">Log Achievement</p>
            <form className="stackForm" onSubmit={handleCreateAchievement}>
              <label>
                Quarter
                <select
                  onChange={(event) => setAchievementForm((current) => ({ ...current, quarter: event.target.value }))}
                  value={achievementForm.quarter}
                >
                  {visibleQuarterGroups.map((group) => (
                    <option key={group.quarter} value={group.quarter}>
                      {group.quarter}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                KPI
                <select
                  onChange={(event) => setAchievementForm((current) => ({ ...current, kpiId: event.target.value }))}
                  value={achievementForm.kpiId}
                >
                  <option value="">Unmapped</option>
                  {kpiOptions
                    .filter((option) => option.quarter === achievementForm.quarter)
                    .map((option) => (
                      <option key={option.id} value={option.id}>
                        {option.title}
                      </option>
                    ))}
                </select>
              </label>
              <label>
                Achievement
                <textarea
                  onChange={(event) => setAchievementForm((current) => ({ ...current, rawText: event.target.value }))}
                  rows={4}
                  value={achievementForm.rawText}
                />
              </label>
              <button disabled={isPending} type="submit">
                Save achievement
              </button>
            </form>
          </article>

          <article className="panel dangerPanel">
            <p className="eyebrow">Protected Reset</p>
            <h3>Reset all progress</h3>
            <p className="mutedText">This clears KPI progress, achievements, reports, and queued jobs for the signed-in user.</p>
            <button className="dangerButton" onClick={() => setResetOpen(true)} type="button">
              Reset all progress
            </button>
          </article>
          {error ? <p className="errorBanner">{error}</p> : null}
        </aside>
      </section>

      {confirmState ? (
        <Modal
          confirmLabel="Confirm action"
          onCancel={() => setConfirmState(null)}
          onConfirm={handleConfirmAction}
          title={confirmState.title}
        >
          <p>{confirmState.message}</p>
        </Modal>
      ) : null}

      {resetOpen ? (
        <Modal
          confirmDisabled={resetConfirmation !== "RESET" || resetPassword.length === 0}
          confirmLabel="Verify password and reset"
          formId="reset-progress-form"
          onCancel={() => setResetOpen(false)}
          title="Reset all progress"
        >
          <form id="reset-progress-form" onSubmit={handleResetProgress}>
            <p>Type <strong>RESET</strong> and re-enter your password before the server clears progress data.</p>
            <label>
              Confirmation text
              <input onChange={(event) => setResetConfirmation(event.target.value)} value={resetConfirmation} />
            </label>
            <label>
              Password
              <input onChange={(event) => setResetPassword(event.target.value)} type="password" value={resetPassword} />
            </label>
          </form>
        </Modal>
      ) : null}
    </main>
  );
}

function flatten(items: KPI[]): KPI[] {
  return items.flatMap((item) => [item, ...flatten(item.children ?? [])]);
}

function KpiTree({
  achievementsByKpi,
  onDelete,
  onPrepareSubKpi,
  quarter,
  reportQuarter,
  root,
}: {
  achievementsByKpi: Map<number, Achievement[]>;
  onDelete: (id: number) => void;
  onPrepareSubKpi: (parent: KPI) => void;
  quarter: string;
  reportQuarter: string;
  root: KPI;
}) {
  const progress = quarterProgressForKpi(root);
  const linkedAchievements = quarter === reportQuarter ? achievementsByKpi.get(root.id) ?? [] : [];

  return (
    <article className="kpiNode">
      <div className="kpiNodeHeader">
        <div>
          <div className="row">
            <h4>{root.title}</h4>
            <span className="progressTag">{progress}%</span>
          </div>
          <p className="mutedText">{root.description || root.targetMetric || "No KPI notes yet."}</p>
        </div>
        <div className="nodeActions">
          <button className="ghost" onClick={() => onPrepareSubKpi(root)} type="button">
            Add sub-KPI
          </button>
          <button className="ghost danger" onClick={() => onDelete(root.id)} type="button">
            Delete
          </button>
        </div>
      </div>
      <div className="progressBar">
        <span style={{ width: `${progress}%` }} />
      </div>
      {linkedAchievements.length > 0 ? (
        <ul className="achievementList">
          {linkedAchievements.map((item) => (
            <li key={item.id}>{item.enhancedText || item.rawText}</li>
          ))}
        </ul>
      ) : null}
      {root.children && root.children.length > 0 ? (
        <div className="subTree">
          {root.children.map((child) => (
            <KpiTree
              achievementsByKpi={achievementsByKpi}
              key={child.id}
              onDelete={onDelete}
              onPrepareSubKpi={onPrepareSubKpi}
              quarter={quarter}
              reportQuarter={reportQuarter}
              root={child}
            />
          ))}
        </div>
      ) : null}
    </article>
  );
}

function ReportTree({ node, reportAchievements }: { node: KPI; reportAchievements: Map<number, Achievement[]> }) {
  const progress = quarterProgressForKpi(node);
  const linkedAchievements = reportAchievements.get(node.id) ?? [];

  return (
    <section className="reportBlock">
      <div className="row">
        <h4>{node.title}</h4>
        <span className="progressTag">{progress}%</span>
      </div>
      {linkedAchievements.length > 0 ? (
        <ul>
          {linkedAchievements.map((item) => (
            <li key={item.id}>{item.enhancedText || item.rawText}</li>
          ))}
        </ul>
      ) : (
        <p className="mutedText">No achievement entries mapped to this KPI for the selected quarter.</p>
      )}
      {node.children && node.children.length > 0 ? (
        <div className="reportChildren">
          {node.children.map((child) => (
            <ReportTree key={child.id} node={child} reportAchievements={reportAchievements} />
          ))}
        </div>
      ) : null}
    </section>
  );
}

function Modal({
  children,
  confirmDisabled,
  confirmLabel,
  formId,
  onCancel,
  onConfirm,
  title,
}: {
  children: ReactNode;
  confirmDisabled?: boolean;
  confirmLabel: string;
  formId?: string;
  onCancel: () => void;
  onConfirm?: () => void;
  title: string;
}) {
  return (
    <div className="modalBackdrop" role="presentation">
      <section aria-modal="true" className="modalCard" role="dialog">
        <div className="sectionHeader">
          <h3>{title}</h3>
          <button className="ghost" onClick={onCancel} type="button">
            Close
          </button>
        </div>
        <div className="modalBody">{children}</div>
        <div className="modalActions">
          <button className="ghost" onClick={onCancel} type="button">
            Cancel
          </button>
          {formId ? (
            <button disabled={confirmDisabled} form={formId} type="submit">
              {confirmLabel}
            </button>
          ) : (
            <button disabled={confirmDisabled} onClick={onConfirm} type="button">
              {confirmLabel}
            </button>
          )}
        </div>
      </section>
    </div>
  );
}
