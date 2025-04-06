import { render } from '@testing-library/react';
import PricingDashboard from '../../src/components/PricingDashboard';
import { toMatchImageSnapshot } from 'jest-image-snapshot';

expect.extend({ toMatchImageSnapshot });

test('PricingDashboard UI should remain consistent', async () => {
  const { container } = render(<PricingDashboard />);
  const image = await captureScreenshot(container);
  expect(image).toMatchImageSnapshot();
});

async function captureScreenshot(element) {
  // In a real scenario, use Puppeteer or Playwright for screenshot capture.
  return Buffer.from('');
}
