// Simulates API failures to validate fallback paths in development.
export const simulateFailure = (config) => {
  if (Math.random() < 0.1) { // 10% chance to simulate failure.
    throw new Error('Simulated API failure for chaos testing');
  }
  return config;
};
