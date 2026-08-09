import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';

const pages = [
  { path: '', heading: /Call any Azure API/i },
  { path: 'getting-started', heading: 'Getting Started' },
  { path: 'examples', heading: 'Examples' },
  { path: 'reference', heading: 'CLI Reference' },
  { path: 'builder', heading: 'Command Builder' },
];
const themes: Array<'light' | 'dark'> = ['light', 'dark'];
const appOrigin = 'http://127.0.0.1:41777';

for (const sitePage of pages) {
  for (const theme of themes) {
    test(`${sitePage.path || 'home'} renders accessibly in ${theme} mode`, async ({ page }) => {
      const runtimeErrors: string[] = [];
      const isFirstParty = (url: string) => new URL(url).origin === appOrigin;

      page.on('console', (message) => {
        if (message.type() === 'error') {
          runtimeErrors.push(`console: ${message.text()}`);
        }
      });
      page.on('pageerror', (error) => runtimeErrors.push(`page: ${error.message}`));
      page.on('requestfailed', (request) => {
        if (isFirstParty(request.url())) {
          runtimeErrors.push(`request: ${request.url()} (${request.failure()?.errorText ?? 'unknown error'})`);
        }
      });
      page.on('response', (response) => {
        if (isFirstParty(response.url()) && response.status() >= 400) {
          runtimeErrors.push(`response: ${response.url()} (${response.status()})`);
        }
      });

      await page.emulateMedia({ colorScheme: theme });
      const response = await page.goto(sitePage.path);
      if (response === null) {
        throw new Error(`Navigation to ${sitePage.path || 'home'} returned no response`);
      }

      expect(response.status()).toBe(200);
      await page.waitForLoadState('networkidle');
      await expect(page.getByRole('heading', { level: 1, name: sitePage.heading })).toBeVisible();
      await expect.poll(() => page.locator('pre, .table-wrapper').evaluateAll((nodes) =>
        nodes.every((node) => node.scrollWidth <= node.clientWidth || node.tabIndex >= 0),
      )).toBe(true);

      const accessibilityScan = await new AxeBuilder({ page })
        .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
        .analyze();
      const violations = accessibilityScan.violations.map((violation) => ({
        id: violation.id,
        impact: violation.impact,
        targets: violation.nodes.flatMap((node) => node.target),
      }));
      expect(violations).toEqual([]);
      expect(runtimeErrors).toEqual([]);
    });
  }
}

test('command builder updates the generated command', async ({ page }) => {
  await page.goto('builder');
  await page.getByLabel('Method').selectOption('post');
  await page.getByLabel('URL').fill('https://management.azure.com/providers');
  await page.getByLabel('Request body').fill('{"name":"example"}');

  const command = page.locator('#generated-command');
  await expect(command).toContainText("azd rest post 'https://management.azure.com/providers");
  await expect(command).toContainText('--data');
  await expect(command).toContainText('{"name":"example"}');
});
