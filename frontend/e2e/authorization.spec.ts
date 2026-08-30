import { expect, test } from '@playwright/test';
import { login } from './support/auth';
import { adminCredentials, professorCredentials, requireBackend, requireCredentials } from './support/env';
import { openNav } from './support/ui';

const expectedHeadings: Record<string, string | RegExp> = {
  Dashboard: /Olá,/,
  Agenda: 'Agenda',
  Treinos: 'Treinos',
  Blocos: 'Blocos',
  Categorias: 'Categorias',
  Histórico: 'Histórico',
  Alunos: 'Alunos',
};

test('ADMIN can access Professores, Alunos and operational areas', async ({ page }) => {
  await requireBackend(page);
  const credentials = adminCredentials();
  requireCredentials(credentials, 'ADMIN');

  await login(page, credentials);
  await openNav(page, 'Professores');
  await expect(page).toHaveURL(/\/app\/professores$/);
  await expect(page.getByRole('heading', { name: 'Professores' })).toBeVisible();

  for (const item of ['Dashboard', 'Agenda', 'Treinos', 'Blocos', 'Categorias', 'Histórico', 'Alunos']) {
    await openNav(page, item);
    await expect(page.getByRole('heading', { name: expectedHeadings[item] })).toBeVisible();
  }
});

test('PROFESSOR cannot access Professores but can access Alunos and operational areas', async ({ page }) => {
  await requireBackend(page);
  const credentials = professorCredentials();
  requireCredentials(credentials, 'PROFESSOR');

  await login(page, credentials);
  await expect(page.locator('.sidebar').getByRole('link', { name: 'Professores' })).toHaveCount(0);

  await page.goto('/app/professores');
  await expect(page).not.toHaveURL(/\/app\/professores$/);

  for (const item of ['Dashboard', 'Agenda', 'Treinos', 'Blocos', 'Categorias', 'Histórico', 'Alunos']) {
    await openNav(page, item);
    await expect(page.getByRole('heading', { name: expectedHeadings[item] })).toBeVisible();
  }
});
