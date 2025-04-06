import { render, waitFor } from '@testing-library/react';
import PricingDashboard from '../../src/components/PricingDashboard';
import { server } from '../mocks/server';
import { rest } from 'msw';

test('Circuit breaker fallback activation on API failure', async () => {
  server.use(
    rest.post('/pricing/calculate', (req, res, ctx) => {
      return res(ctx.delay(10000)); // Delay to simulate timeout
    })
  );

  const { getByRole } = render(<PricingDashboard />);
  await waitFor(() => {
    expect(getByRole('alert')).toHaveTextContent(/fallback/i);
  });
});
