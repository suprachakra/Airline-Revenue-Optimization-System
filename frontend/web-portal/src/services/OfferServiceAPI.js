import api from './apiClient';

export const fetchOfferData = async (userId) => {
  try {
    const response = await api.get(`/offer/${userId}`);
    return response.data;
  } catch (error) {
    console.error('Offer service API error:', error);
    return getCachedOffer(userId);
  }
};

function getCachedOffer(userId) {
  // In production, query a secure cache for offers.
  return { offers: [], source: 'cache' };
}
