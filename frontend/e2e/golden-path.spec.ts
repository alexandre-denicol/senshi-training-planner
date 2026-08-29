import { expect, Page, test } from '@playwright/test';
import { login } from './support/auth';
import {
  adminCredentials,
  brDateFromISO,
  isoDateInCurrentMonth,
  requireBackend,
  requireCredentials,
  requireMutationSafety,
  uniqueName,
} from './support/env';
import { dialog, expectTextInOrder, openNav } from './support/ui';

test('creates catalog data, schedules, completes training and verifies immutable History snapshot', async ({ page }) => {
  await requireBackend(page);
  requireMutationSafety();
  const credentials = adminCredentials();
  requireCredentials(credentials, 'ADMIN');

  const categoryName = uniqueName('Categoria Golden');
  const originalBlockName = uniqueName('Bloco Golden');
  const editedBlockName = `${originalBlockName} Editado`;
  const workoutName = uniqueName('Treino Golden');
  const scheduledDate = isoDateInCurrentMonth();
  const displayDate = brDateFromISO(scheduledDate);
  const originalDescription = 'Treino realizado pelo teste E2E.\nDescrição original do bloco.';
  const editedDescription = 'Descrição editada depois da conclusão.';
  const originalSequence = ['Jab', 'Direto', 'Hook frente', 'Mawashi geri trás'];
  const editedSequence = ['Sprawl', 'Trocar base'];

  await login(page, credentials);

  await createCategory(page, categoryName);
  await createBlock(page, originalBlockName, categoryName, originalDescription, originalSequence);
  await createWorkout(page, workoutName, originalBlockName);
  await scheduleWorkout(page, workoutName, scheduledDate);

  await openNav(page, 'Agenda');
  const agendaEntry = page.locator('.entry-card').filter({ hasText: workoutName }).first();
  await expect(agendaEntry).toContainText(workoutName);
  await agendaEntry.getByRole('button', { name: 'Ver treino' }).click();
  const details = dialog(page, 'Detalhes do treino');
  await expect(details).toContainText(workoutName);
  await expect(details).toContainText(originalBlockName);
  await expect(details).toContainText(categoryName);
  await expect(details).toContainText(originalDescription);
  await expect(details).toContainText(originalSequence.join(' → '));
  await details.getByRole('button', { name: 'Fechar' }).click();

  await agendaEntry.getByRole('button', { name: 'Realizar treino' }).click();
  const completion = dialog(page, 'Registrar treino realizado');
  await completion.locator('input[name="participant-count"]').fill('3');
  for (const participant of ['Aluno E2E 1', 'Aluno E2E 2', 'Aluno E2E 3']) {
    await completion.getByLabel('Nome do participante').fill(participant);
    await completion.getByRole('button', { name: 'Adicionar' }).click();
  }
  await completion.locator('textarea[name="completion-notes"]').fill('Treino realizado pelo teste E2E.');
  await completion.getByRole('button', { name: 'Registrar como realizado' }).click();
  await expect(agendaEntry).toContainText('Realizado');

  await verifyHistorySnapshot(page, {
    workoutName,
    blockName: originalBlockName,
    categoryName,
    description: originalDescription,
    sequence: originalSequence,
    participantCount: 'Quantidade: 3 alunos',
    participants: ['Aluno E2E 1', 'Aluno E2E 2', 'Aluno E2E 3'],
    notes: 'Treino realizado pelo teste E2E.',
    displayDate,
  });

  await editBlock(page, originalBlockName, editedBlockName, editedDescription, editedSequence);

  await verifyHistorySnapshot(page, {
    workoutName,
    blockName: originalBlockName,
    categoryName,
    description: originalDescription,
    sequence: originalSequence,
    participantCount: 'Quantidade: 3 alunos',
    participants: ['Aluno E2E 1', 'Aluno E2E 2', 'Aluno E2E 3'],
    notes: 'Treino realizado pelo teste E2E.',
    displayDate,
    absentText: editedBlockName,
  });
});

async function createCategory(page: Page, categoryName: string): Promise<void> {
  await openNav(page, 'Categorias');
  await page.getByRole('button', { name: 'Nova categoria' }).first().click();
  const modal = dialog(page, 'Nova categoria');
  await modal.getByLabel('Nome').fill(categoryName);
  await modal.getByRole('button', { name: 'Salvar' }).click();
  await expect(page.getByRole('status')).toContainText('Categoria cadastrada com sucesso.');
}

async function createBlock(page: Page, blockName: string, categoryName: string, description: string, sequence: string[]): Promise<void> {
  await openNav(page, 'Blocos');
  await page.getByRole('button', { name: 'Novo bloco' }).first().click();
  const modal = dialog(page, 'Novo bloco');
  await modal.getByLabel('Nome').fill(blockName);
  await modal.getByLabel('Categoria').selectOption({ label: categoryName });
  await modal.getByLabel('Descrição / instruções (opcional)').fill(description);
  for (const item of sequence) {
    await modal.getByLabel('Item da sequência').fill(item);
    await modal.getByRole('button', { name: 'Adicionar' }).click();
  }
  await expectTextInOrder(modal.locator('.sequence-list'), sequence);
  await modal.getByRole('button', { name: 'Salvar' }).click();
  await expect(page.getByRole('status')).toContainText('Bloco cadastrado com sucesso.');
}

async function createWorkout(page: Page, workoutName: string, blockName: string): Promise<void> {
  await openNav(page, 'Treinos');
  await page.getByRole('button', { name: 'Novo treino' }).first().click();
  const modal = dialog(page, 'Novo treino');
  await modal.getByLabel('Nome').fill(workoutName);
  await modal.getByRole('button', { name: 'Adicionar bloco' }).click();
  const addBlock = dialog(page, 'Adicionar bloco');
  await addBlock.getByLabel('Buscar').fill(blockName);
  const option = addBlock.locator('.block-option').filter({ hasText: blockName }).first();
  await expect(option).toContainText('Jab → Direto → Hook frente → Mawashi geri trás');
  await option.getByRole('button', { name: 'Adicionar' }).click();
  await addBlock.getByRole('button', { name: 'Concluir' }).click();
  await expect(modal.locator('.selected-blocks')).toContainText(blockName);
  await modal.getByRole('button', { name: 'Salvar' }).click();
  await expect(dialog(page, 'Treino criado com sucesso')).toBeVisible();
}

async function scheduleWorkout(page: Page, workoutName: string, scheduledDate: string): Promise<void> {
  const postSave = dialog(page, 'Treino criado com sucesso');
  await expect(postSave).toContainText(workoutName);
  await postSave.locator('input[name="created-workout-date"]').fill(scheduledDate);
  await postSave.getByRole('button', { name: 'Agendar este treino' }).click();
  await expect(page.getByRole('status')).toContainText('Treino agendado com sucesso.');
}

async function editBlock(page: Page, oldName: string, newName: string, description: string, sequence: string[]): Promise<void> {
  await openNav(page, 'Blocos');
  await page.getByRole('row', { name: new RegExp(oldName) }).getByRole('button', { name: 'Editar bloco' }).click();
  const modal = dialog(page, 'Editar bloco');
  await modal.getByLabel('Nome').fill(newName);
  await modal.getByLabel('Descrição / instruções (opcional)').fill(description);
  while (await modal.getByRole('button', { name: 'Remover item' }).count() > 0) {
    await modal.getByRole('button', { name: 'Remover item' }).first().click();
  }
  for (const item of sequence) {
    await modal.getByLabel('Item da sequência').fill(item);
    await modal.getByRole('button', { name: 'Adicionar' }).click();
  }
  await modal.getByRole('button', { name: 'Salvar' }).click();
  await expect(page.getByRole('status')).toContainText('Bloco atualizado com sucesso.');
}

async function verifyHistorySnapshot(page: Page, expected: {
  workoutName: string;
  blockName: string;
  categoryName: string;
  description: string;
  sequence: string[];
  participantCount: string;
  participants: string[];
  notes: string;
  displayDate: string;
  absentText?: string;
}): Promise<void> {
  await openNav(page, 'Histórico');
  const card = page.locator('.history-card').filter({ hasText: expected.workoutName }).first();
  await expect(card).toBeVisible();
  await card.click();
  const history = dialog(page, 'Detalhes do histórico');
  await expect(history).toContainText(expected.workoutName);
  await expect(history).toContainText(expected.displayDate);
  await expect(history).toContainText(expected.blockName);
  await expect(history).toContainText(expected.categoryName);
  await expect(history).toContainText(expected.description);
  await expectTextInOrder(history.locator('.snapshot-sequence'), expected.sequence);
  await expect(history).toContainText(expected.participantCount);
  for (const participant of expected.participants) {
    await expect(history).toContainText(participant);
  }
  await expect(history).toContainText(expected.notes);
  if (expected.absentText) {
    await expect(history).not.toContainText(expected.absentText);
  }
  await page.keyboard.press('Escape');
}
