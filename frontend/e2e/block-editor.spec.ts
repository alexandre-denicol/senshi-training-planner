import { expect, test } from '@playwright/test';
import { login } from './support/auth';
import { adminCredentials, requireBackend, requireCredentials, requireMutationSafety, uniqueName } from './support/env';
import { dialog, expectTextInOrder, openNav } from './support/ui';

test('richer Block editor supports free-text description and ordered sequence', async ({ page }) => {
  await requireBackend(page);
  requireMutationSafety();
  const credentials = adminCredentials();
  requireCredentials(credentials, 'ADMIN');

  const categoryName = uniqueName('Categoria Blocos');
  const blockName = uniqueName('Bloco Livre');
  const description = 'Executar em dupla.\nUm atleta ataca enquanto o outro responde.';

  await login(page, credentials);

  await openNav(page, 'Categorias');
  await page.getByRole('button', { name: 'Nova categoria' }).first().click();
  const categoryDialog = dialog(page, 'Nova categoria');
  await categoryDialog.getByLabel('Nome').fill(categoryName);
  await categoryDialog.getByRole('button', { name: 'Salvar' }).click();
  await expect(page.getByRole('status')).toContainText('Categoria cadastrada com sucesso.');

  await openNav(page, 'Blocos');
  await page.getByRole('button', { name: 'Novo bloco' }).first().click();
  const createDialog = dialog(page, 'Novo bloco');
  await createDialog.getByLabel('Nome').fill(blockName);
  await createDialog.getByLabel('Categoria').selectOption({ label: categoryName });
  await createDialog.getByLabel('Descrição / instruções (opcional)').fill(description);
  await createDialog.getByLabel('Item da sequência').fill('Jab');
  await createDialog.getByRole('button', { name: 'Adicionar' }).click();
  await expect(createDialog.getByLabel('Item da sequência')).toHaveValue('');
  await createDialog.getByLabel('Item da sequência').fill('Direto');
  await createDialog.getByLabel('Item da sequência').press('Enter');
  await createDialog.getByLabel('Item da sequência').fill('Direto');
  await createDialog.getByRole('button', { name: 'Adicionar' }).click();
  await createDialog.getByLabel('Item da sequência').fill('Mawashi geri trás');
  await createDialog.getByRole('button', { name: 'Adicionar' }).click();

  await expectTextInOrder(createDialog.locator('.sequence-list'), ['Jab', 'Direto', 'Direto', 'Mawashi geri trás']);
  await createDialog.getByRole('button', { name: 'Mover item para cima' }).nth(3).click();
  await createDialog.getByRole('button', { name: 'Mover item para baixo' }).first().click();
  await createDialog.getByRole('button', { name: 'Remover item' }).nth(2).click();
  await expectTextInOrder(createDialog.locator('.sequence-list'), ['Jab', 'Mawashi geri trás', 'Direto']);
  await createDialog.getByRole('button', { name: 'Salvar' }).click();

  await expect(page.getByRole('status')).toContainText('Bloco cadastrado com sucesso.');
  await page.getByRole('row', { name: new RegExp(blockName) }).getByRole('button', { name: 'Editar bloco' }).click();
  const editDialog = dialog(page, 'Editar bloco');
  await expect(editDialog.getByLabel('Descrição / instruções (opcional)')).toHaveValue(description);
  await expectTextInOrder(editDialog.locator('.sequence-list'), ['Jab', 'Mawashi geri trás', 'Direto']);
});
