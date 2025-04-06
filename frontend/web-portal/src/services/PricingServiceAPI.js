import api from './apiClient';

export const fetchDynamicPrice = async (route, parameters) => {
  try {
    const response = await api.post('/pricing/calculate', { route, parameters });
    return response.data;
  } catch (error) {
    console.error('Dynamic pricing API error:', error);
    return getCachedPrice(route);
  }
};

function getCachedPrice(route) {
  // In production, retrieve from a secure cache service.
  return { price: 100.0, source: 'cache' };
}
