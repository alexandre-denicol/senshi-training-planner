import { expect, test } from '@playwright/test';
import { login, logout } from './support/auth';
import { adminCredentials, professorCredentials, requireBackend, requireCredentials } from './support/env';

test('unauthenticated user is redirected from protected app routes', async ({ page }) => {
  await page.goto('/app/blocos');
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole('heading', { name: 'Catálogo e planejamento de treinos' })).toBeVisible();
});

test('invalid login shows generic authentication feedback', async ({ page }) => {
  await requireBackend(page);

  await page.goto('/login');
  await page.getByLabel('E-mail').fill(`invalid-${Date.now()}@example.test`);
  await page.getByLabel('Senha').fill('senha incorreta deliberadamente longa');
  await page.getByRole('button', { name: 'Entrar' }).click();

  await expect(page.getByRole('alert')).toContainText('E-mail ou senha inválidos.');
});

test('valid ADMIN login and logout use the real session flow', async ({ page }) => {
  await requireBackend(page);
  const credentials = adminCredentials();
  requireCredentials(credentials, 'ADMIN');

  await login(page, credentials);
  await expect(page.getByText(/Olá,/)).toBeVisible();
  await logout(page);
});

test('valid PROFESSOR login and logout use the real session flow', async ({ page }) => {
  await requireBackend(page);
  const credentials = professorCredentials();
  requireCredentials(credentials, 'PROFESSOR');

  await login(page, credentials);
  await expect(page.getByText(/Olá,/)).toBeVisible();
  await logout(page);
});
