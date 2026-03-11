import type { Report } from "./api";

export type ReportFormat = "review" | "notion" | "hr";

export function formatReport(report: Report | null, format: ReportFormat): string {
  if (!report) {
    return "No report generated yet.";
  }

  switch (format) {
    case "notion":
      return `# ${report.title}\n\n${report.body
        .split("\n")
        .map((line) => (line.startsWith("- ") ? line.replace("- ", "* ") : line))
        .join("\n")}`;
    case "hr":
      return `Quarter: ${report.quarter}\nDocument: ${report.title}\n\nSummary for performance review:\n${report.body}`;
    default:
      return report.body;
  }
}
