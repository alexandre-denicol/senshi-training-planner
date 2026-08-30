import { expect, Locator, Page, test } from '@playwright/test';
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

test('creates catalog data, schedules, completes training with registered Student participants and verifies immutable History snapshot', async ({ page }) => {
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
  const studentOriginalName = uniqueName('Aluno Golden');
  const studentRenamedName = `${studentOriginalName} Renomeado`;
  const studentControlName = uniqueName('Aluno Golden Controle');

  await login(page, credentials);

  await createStudent(page, studentOriginalName);
  await createStudent(page, studentControlName);
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
  await selectParticipants(completion, [studentOriginalName, studentControlName]);
  await expect(completion).toContainText('Participantes selecionados (2)');
  await completion.locator('textarea[name="completion-notes"]').fill('Treino realizado pelo teste E2E.');
  await completion.getByRole('button', { name: 'Registrar como realizado' }).click();
  await expect(agendaEntry).toContainText('Realizado');

  const expectedSnapshot = {
    workoutName,
    blockName: originalBlockName,
    categoryName,
    description: originalDescription,
    sequence: originalSequence,
    participantCount: 'Quantidade: 2 alunos',
    participants: [studentOriginalName, studentControlName],
    notes: 'Treino realizado pelo teste E2E.',
    displayDate,
  };

  await verifyHistorySnapshot(page, expectedSnapshot);

  await editBlock(page, originalBlockName, editedBlockName, editedDescription, editedSequence);

  await verifyHistorySnapshot(page, { ...expectedSnapshot, absentText: editedBlockName });

  // Rename the participant Student after the training is already completed. The
  // History record must keep showing the original name forever - it is an
  // immutable snapshot, not a live lookup into the Students table.
  await renameStudent(page, studentOriginalName, studentRenamedName);

  await page.reload();

  await verifyHistorySnapshot(page, {
    ...expectedSnapshot,
    absentText: studentRenamedName,
  });
});

async function createStudent(page: Page, name: string): Promise<void> {
  await openNav(page, 'Alunos');
  await page.getByRole('button', { name: 'Novo aluno' }).first().click();
  const modal = dialog(page, 'Novo aluno');
  await modal.getByLabel('Nome').fill(name);
  await modal.getByRole('button', { name: 'Salvar' }).click();
  await expect(page.getByRole('status')).toContainText('Aluno cadastrado com sucesso.');
  await expect(page.getByRole('row', { name: new RegExp(name) })).toBeVisible();
}

async function renameStudent(page: Page, currentName: string, newName: string): Promise<void> {
  await openNav(page, 'Alunos');
  await page.getByRole('row', { name: new RegExp(currentName) }).getByRole('button', { name: 'Editar aluno' }).click();
  const modal = dialog(page, 'Editar aluno');
  await modal.getByLabel('Nome').fill(newName);
  await modal.getByRole('button', { name: 'Salvar' }).click();
  await expect(page.getByRole('status')).toContainText('Aluno atualizado com sucesso.');
  await expect(page.getByRole('row', { name: new RegExp(newName) })).toBeVisible();
}

async function selectParticipants(completion: Locator, names: string[]): Promise<void> {
  for (const name of names) {
    await completion.getByLabel('Buscar aluno pelo nome').fill(name);
    await completion.getByRole('button', { name, exact: true }).click();
  }
}

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
  const removeButtons = modal.getByRole('button', { name: 'Remover item' });
  while ((await removeButtons.count()) > 0) {
    const count = await removeButtons.count();
    await removeButtons.nth(count - 1).click();
    await expect(removeButtons).toHaveCount(count - 1);
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
