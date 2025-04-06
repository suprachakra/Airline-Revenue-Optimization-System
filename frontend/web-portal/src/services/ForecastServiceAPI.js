import api from './apiClient';

export const fetchForecastData = async (route, dateRange) => {
  try {
    const response = await api.get(`/forecast/${route}`, { params: { dateRange } });
    return response.data;
  } catch (error) {
    console.error('Forecast API error:', error);
    return getCachedForecast(route);
  }
};

function getCachedForecast(route) {
  // In production, retrieve from secure persistent cache.
  return { forecast: [{ date: '2025-01-01', value: 120 }, { date: '2025-01-02', value: 115 }], source: 'cache' };
}
