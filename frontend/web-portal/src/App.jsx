import React from 'react';
import { BrowserRouter as Router, Route, Switch } from 'react-router-dom';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import PricingControlsPage from './pages/PricingControlsPage';
import ReportsPage from './pages/ReportsPage';
import GlobalErrorBoundary from './components/GlobalErrorBoundary';
import { GDPRGuard } from './utils/compliance/GDPRGuard';

export default function App() {
  return (
    <GlobalErrorBoundary>
      <GDPRGuard>
        <Router>
          <Switch>
            <Route exact path="/" component={DashboardPage} />
            <Route path="/login" component={LoginPage} />
            <Route path="/pricing-controls" component={PricingControlsPage} />
            <Route path="/reports" component={ReportsPage} />
          </Switch>
        </Router>
      </GDPRGuard>
    </GlobalErrorBoundary>
  );
}
