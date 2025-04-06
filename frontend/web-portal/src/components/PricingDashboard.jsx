import React, { useEffect } from 'react';
import { usePricingData } from '../hooks/usePricingData';
import { formatTimeAgo } from '../utils/timeUtils';
import SkeletonDashboard from './SkeletonDashboard';
import Alert from '@mui/material/Alert';
import HistoricalPricingChart from './HistoricalPricingChart';
import { usePersistentCache } from '../hooks/usePersistentCache';
import RealTimePricingChart from './RealTimePricingChart';
import RefreshControl from './RefreshControl';
import { ErrorBoundary } from './GlobalErrorBoundary';

export default function PricingDashboard() {
  const { data, error, status } = usePricingData();
  const [cachedData, setCachedData] = usePersistentCache('pricing_data');

  useEffect(() => {
    if (status === 'success') {
      setCachedData({ ...data, _timestamp: Date.now() });
    }
  }, [status, data, setCachedData]);

  return (
    <ErrorBoundary FallbackComponent={NuclearFallbackUI}>
      {status === 'loading' && <SkeletonDashboard />}
      {status === 'error' && <CachedDataView data={cachedData} />}
      {status === 'success' && <RealTimePricingChart data={data} />}
    </ErrorBoundary>
  );
}

const CachedDataView = ({ data }) => {
  const isStale = Date.now() - data._timestamp > 300000; // 5-minute threshold
  return (
    <div className="alert-container">
      <Alert severity={isStale ? 'warning' : 'info'}>
        {isStale ? 'Data may be outdated; showing cached results.' : 'Using cached data.'}
        {' '}Last updated {formatTimeAgo(data._timestamp)}.
      </Alert>
      <HistoricalPricingChart data={data} />
      <RefreshControl onRetry={() => window.location.reload()} />
    </div>
  );
};
