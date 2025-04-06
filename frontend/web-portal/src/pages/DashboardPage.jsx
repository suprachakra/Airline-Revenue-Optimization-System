import React from 'react';
import PricingDashboard from '../components/PricingDashboard';
import ForecastChart from '../components/ForecastChart';
import ReportsPage from './ReportsPage';

export default function DashboardPage() {
  return (
    <div className="dashboard-page">
      <header>
        <h1>Dashboard</h1>
      </header>
      <section>
        <PricingDashboard />
        <ForecastChart />
      </section>
      <section>
        <ReportsPage />
      </section>
    </div>
  );
}
