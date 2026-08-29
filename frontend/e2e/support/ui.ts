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
  const text = await container.textContent();
  expect(text ?? '').toContain(values[0]);
  let previous = -1;
  for (const value of values) {
    const index = (text ?? '').indexOf(value);
    expect(index, `${value} should be visible after the previous value`).toBeGreaterThan(previous);
    previous = index;
  }
}
