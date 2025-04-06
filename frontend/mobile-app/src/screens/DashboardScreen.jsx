import React from 'react';
import { SafeAreaView, View, Text } from 'react-native';
import { useNetworkStatus } from '@react-native-community/netinfo';
import RealTimeMetrics from '../components/RealTimeMetrics';
import OfflineBanner from '../components/OfflineBanner';
import RefreshControl from '../components/RefreshControl';

export default function DashboardScreen() {
  const { isConnected } = useNetworkStatus();
  const { data, error } = useFetchDashboardData(); // Custom hook for fetching dashboard data

  return (
    <SafeAreaView style={{ flex: 1 }}>
      {!isConnected ? (
        <OfflineBanner />
      ) : error ? (
        <FallbackDashboard />
      ) : (
        <RealTimeMetrics data={data} />
      )}
      <RefreshControl onRefresh={handleRefresh} fallbackMessage="Data refresh paused - network issues detected" />
    </SafeAreaView>
  );
}

const FallbackDashboard = () => (
  <View style={{ padding: 20 }}>
    <Text style={{ color: 'red' }}>
      Live data unavailable - showing cached values.
    </Text>
    <CachedMetrics />
    <ComplianceWarning />
  </View>
);
