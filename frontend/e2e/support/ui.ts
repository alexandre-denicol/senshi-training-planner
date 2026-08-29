import { expect, Locator, Page } from '@playwright/test';

export async function openNav(page: Page, label: string): Promise<void> {
  const desktopLink = page.locator('.sidebar').getByRole('link', { name: label });
  if (await desktopLink.isVisible()) {
    await desktopLink.click();
    return;
  }

  await page.getByRole('button', { name: 'Abrir navegação' }).click();
  await page.locator('.mobile-drawer').getByRole('link', { name: label }).click();
}

export function dialog(page: Page, name: string): Locator {
  return page.getByRole('dialog', { name });
}

export async function expectTextInOrder(container: Locator, values: string[]): Promise<void> {
  const items = container.locator('li');
  if ((await items.count()) > 0) {
    await expect(items).toHaveCount(values.length);
    for (const [index, value] of values.entries()) {
      await expect(items.nth(index), `${value} should be visible at position ${index + 1}`).toContainText(value);
    }
    return;
  }

  const text = await container.textContent();
  expect(text ?? '').toContain(values[0]);
  let previous = -1;
  for (const value of values) {
    const index = (text ?? '').indexOf(value, previous + 1);
    expect(index, `${value} should be visible after the previous value`).toBeGreaterThan(previous);
    previous = index;
  }
}
