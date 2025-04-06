import { render, waitFor } from '@testing-library/react';
import DashboardPage from '../../src/pages/DashboardPage';
import { simulateFailure } from '../../src/utils/resilience/ChaosInterceptor';

test('API failure triggers nuclear fallback UI', async () => {
  simulateFailure(); // Trigger simulated API failure
  const { getByTestId } = render(<DashboardPage />);
  await waitFor(() => {
    expect(getByTestId('nuclear-fallback-ui')).toBeVisible();
  });
});
