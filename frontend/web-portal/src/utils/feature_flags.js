// Feature flags for gradual rollouts and A/B testing.
export const isFeatureEnabled = (feature) => {
  return window.APP_CONFIG?.featureFlags?.[feature] === true;
};
