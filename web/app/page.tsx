import { Dashboard } from "../components/dashboard";
import { DashboardRedesign } from "../components/dashboard-redesign";

export default function Home() {
  return process.env.NEXT_PUBLIC_NEW_DASHBOARD_ENABLED === "false" ? <Dashboard /> : <DashboardRedesign />;
}
