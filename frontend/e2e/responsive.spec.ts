import { expect, test } from '@playwright/test';
import { login } from './support/auth';
import { adminCredentials, requireBackend, requireCredentials, requireMutationSafety } from './support/env';
import { dialog, openNav } from './support/ui';

test.use({ viewport: { width: 390, height: 844 } });

test('mobile smoke covers navigation, Agenda, Block editor and Workout Builder', async ({ page }) => {
  await requireBackend(page);
  requireMutationSafety();
  const credentials = adminCredentials();
  requireCredentials(credentials, 'ADMIN');

  await login(page, credentials);
  await expect(page.getByRole('button', { name: 'Abrir navegação' })).toBeVisible();

  await openNav(page, 'Dashboard');
  await expect(page.getByText(/Olá,/)).toBeVisible();

  await openNav(page, 'Agenda');
  await expect(page.getByRole('heading', { name: 'Agenda' })).toBeVisible();

  await openNav(page, 'Blocos');
  await expect(page.getByRole('heading', { name: 'Blocos' })).toBeVisible();
  const newBlockButton = page.getByRole('button', { name: 'Novo bloco' }).first();
  if (await newBlockButton.isEnabled()) {
    await newBlockButton.click();
    await expect(dialog(page, 'Novo bloco')).toBeVisible();
    await page.keyboard.press('Escape');
  }

  await openNav(page, 'Treinos');
  await expect(page.getByRole('heading', { name: 'Treinos' })).toBeVisible();
  await page.getByRole('button', { name: 'Novo treino' }).first().click();
  await expect(dialog(page, 'Novo treino')).toBeVisible();
});
