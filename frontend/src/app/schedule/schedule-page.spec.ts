import { HttpErrorResponse } from '@angular/common/http';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { AuthUser } from '../auth/auth.models';
import { AuthService } from '../auth/auth.service';
import { Student, StudentApiService } from '../students/student-api.service';
import { WorkoutApiService, WorkoutDetail, WorkoutListItem } from '../workouts/workout-api.service';
import { ScheduleApiService, ScheduleEntry } from './schedule-api.service';
import { SchedulePage } from './schedule-page';

const adminUser: AuthUser = {
  id: 'admin-id',
  name: 'Admin',
  email: 'admin@example.com',
  role: 'ADMIN',
};

const professorUser: AuthUser = {
  id: 'professor-id',
  name: 'Professor',
  email: 'professor@example.com',
  role: 'PROFESSOR',
};

describe('SchedulePage', () => {
  it('allows ADMIN access and shows scheduling controls', async () => {
    const { fixture } = await renderPage(adminUser, [], [workout()]);

    expect(fixture.nativeElement.textContent).toContain('Agenda');
    expect(fixture.nativeElement.textContent).toContain('Agendar treino');
  });

  it('allows PROFESSOR operational access with schedule and completion controls', async () => {
    const workoutApi = new FakeWorkoutApi([workout()]);
    const { fixture } = await renderPage(professorUser, [entry()], [workout()], undefined, workoutApi);
    const text = fixture.nativeElement.textContent;

    expect(text).toContain('Treino Base');
    expect(text).toContain('Agendar treino');
    expect(text).toContain('Ver treino');
    expect(text).toContain('Realizar treino');
    expect(fixture.nativeElement.querySelector('button[aria-label="Editar agendamento"]')).toBeTruthy();
    expect(fixture.nativeElement.querySelector('button[aria-label="Remover agendamento"]')).toBeTruthy();
    expect(workoutApi.listCalls).toBeGreaterThan(0);
  });

  it('shows empty state', async () => {
    const { fixture } = await renderPage(adminUser, [], [workout()]);

    expect(fixture.nativeElement.textContent).toContain('Nenhum treino agendado neste período.');
  });

  it('shows populated date-grouped agenda with multiple workouts on the same date', async () => {
    const entries = [
      entry({ id: 'entry-2', scheduledDate: '2026-08-24', workout: { id: 'workout-2', name: 'Treino B', active: true } }),
      entry({ id: 'entry-1', scheduledDate: '2026-08-24', workout: { id: 'workout-1', name: 'Treino A', active: true } }),
      entry({ id: 'entry-3', scheduledDate: '2026-08-25', workout: { id: 'workout-3', name: 'Treino C', active: true } }),
    ];

    const { fixture } = await renderPage(adminUser, entries, [workout()]);
    const text = fixture.nativeElement.textContent;

    expect(text).toContain('24/08/2026');
    expect(text).toContain('25/08/2026');
    expect(text).toContain('Treino A');
    expect(text).toContain('Treino B');
    expect(text).toContain('Treino C');
  });

  it('opens scheduled workout detail for ADMIN with loading state, ordered blocks and categories', async () => {
    const original = entry({ id: 'entry-1', workout: { id: 'workout-1', name: 'Treino Técnico', active: true } });
    const workoutApi = new FakeWorkoutApi([workout()]);
    workoutApi.detailGate = pendingPromise();
    workoutApi.details.set('workout-1', workoutDetail({
      id: 'workout-1',
      name: 'Treino Técnico',
      blocks: [
        workoutBlock({
          id: 'block-1',
          name: 'Aquecimento',
          description: 'Preparar articulações e elevar frequência.',
          sequence: [
            { position: 1, text: 'Corrida leve' },
            { position: 2, text: 'Mobilidade de quadril' },
          ],
          position: 1,
          category: { id: 'category-1', name: 'Preparação' },
        }),
        workoutBlock({ id: 'block-2', name: 'Combinações de jab e direto com nome bastante longo para validar quebra natural', position: 2, category: { id: 'category-2', name: 'Técnica' } }),
        workoutBlock({ id: 'block-3', name: 'Alongamento', active: false, position: 3, category: { id: 'category-3', name: 'Finalização' } }),
      ],
    }));
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], undefined, workoutApi);

    const loading = component.openWorkoutDetail(original);
    fixture.detectChanges();

    expect(workoutApi.loadedDetails).toEqual(['workout-1']);
    expect(component.workoutDetailDialogOpen()).toBe(true);
    expect(fixture.nativeElement.textContent).toContain('Detalhes do treino');
    expect(fixture.nativeElement.textContent).toContain('Carregando detalhes do treino');

    workoutApi.detailGate.resolve();
    await loading;
    fixture.detectChanges();

    const detailText = detailBodyText(fixture.nativeElement);
    expect(detailText).toContain('Treino Técnico');
    expect(detailText).toContain('24/08/2026');
    expect(detailText).toContain('Blocos do treino');
    expect(detailText).toContain('Aquecimento');
    expect(detailText).toContain('Categoria: Preparação');
    expect(detailText).toContain('Preparar articulações e elevar frequência.');
    expect(detailText).toContain('Corrida leve → Mobilidade de quadril');
    expect(detailText).toContain('Combinações de jab e direto');
    expect(detailText).toContain('Categoria: Técnica');
    expect(detailText).toContain('Alongamento');
    expect(detailText).toContain('Bloco inativo');
    expect(detailText.indexOf('Aquecimento')).toBeLessThan(detailText.indexOf('Combinações de jab'));
    expect(detailText.indexOf('Combinações de jab')).toBeLessThan(detailText.indexOf('Alongamento'));
    expect(detailText).not.toContain('Editar');
    expect(detailText).not.toContain('Excluir');
    expect(detailText).not.toContain('Salvar');
  });

  it('opens scheduled workout detail for PROFESSOR while retaining agenda management data', async () => {
    const original = entry({ id: 'entry-1', workout: { id: 'workout-1', name: 'Treino Técnico', active: true } });
    const workoutApi = new FakeWorkoutApi([]);
    workoutApi.details.set('workout-1', workoutDetail({ id: 'workout-1', name: 'Treino Técnico' }));
    const { fixture, component } = await renderPage(professorUser, [original], [], undefined, workoutApi);

    await component.openWorkoutDetail(original);
    fixture.detectChanges();

    expect(workoutApi.listCalls).toBeGreaterThan(0);
    expect(workoutApi.loadedDetails).toEqual(['workout-1']);
    expect(detailBodyText(fixture.nativeElement)).toContain('Treino Técnico');
  });

  it('prevents repeated workout detail requests while the same entry is loading', async () => {
    const original = entry({ id: 'entry-1', workout: { id: 'workout-1', name: 'Treino Técnico', active: true } });
    const workoutApi = new FakeWorkoutApi([]);
    workoutApi.detailGate = pendingPromise();
    const { component } = await renderPage(adminUser, [original], [workout()], undefined, workoutApi);

    const first = component.openWorkoutDetail(original);
    const second = component.openWorkoutDetail(original);

    expect(workoutApi.loadedDetails).toEqual(['workout-1']);
    workoutApi.detailGate.resolve();
    await first;
    await second;
  });

  it('shows safe workout detail load errors and clears dialog state on close', async () => {
    const original = entry({ id: 'entry-1', workout: { id: 'missing-workout', name: 'Treino Removido', active: true } });
    const workoutApi = new FakeWorkoutApi([]);
    workoutApi.detailError = new HttpErrorResponse({ status: 404, statusText: 'Not Found' });
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], undefined, workoutApi);

    await component.openWorkoutDetail(original);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Treino não encontrado.');
    component.closeWorkoutDetailDialog();

    expect(component.workoutDetailDialogOpen()).toBe(false);
    expect(component.workoutDetail()).toBeNull();
    expect(component.workoutDetailError()).toBe('');
    expect(component.workoutDetailEntry).toBeNull();
  });

  it('formats API DATE values without timezone shifting', async () => {
    const { fixture, component } = await renderPage(adminUser, [entry({ scheduledDate: '2026-08-01' })], [workout()]);

    expect(component.formatDate('2026-08-01')).toBe('01/08/2026');
    expect(fixture.nativeElement.textContent).toContain('01/08/2026');
    expect(fixture.nativeElement.textContent).toContain('agosto');
  });

  it('requests bounded month ranges for navigation', async () => {
    const api = new FakeScheduleApi([]);
    const { component } = await renderPage(adminUser, [], [workout()], api);

    component.currentMonth.set(new Date(2026, 7, 1));
    await component.loadData();
    expect(api.ranges.at(-1)).toEqual({ from: '2026-08-01', to: '2026-08-31' });

    await component.nextMonth();
    expect(api.ranges.at(-1)).toEqual({ from: '2026-09-01', to: '2026-09-30' });

    await component.previousMonth();
    expect(api.ranges.at(-1)).toEqual({ from: '2026-08-01', to: '2026-08-31' });

    await component.goToToday();
    expect(api.ranges.at(-1)?.from).toMatch(/^\d{4}-\d{2}-01$/);
  });

  it('creates agenda entry and handles duplicate scheduling', async () => {
    const api = new FakeScheduleApi([]);
    const { fixture, component } = await renderPage(adminUser, [], [workout({ id: 'workout-1' })], api);

    component.openCreateDialog();
    component.createForm = { workoutId: 'workout-1', scheduledDate: '2026-08-24' };
    await component.createEntry();
    fixture.detectChanges();

    expect(api.created[0]).toEqual({ workoutId: 'workout-1', scheduledDate: '2026-08-24' });
    expect(component.createForm).toEqual({ scheduledDate: '', workoutId: '' });
    expect(fixture.nativeElement.textContent).toContain('Treino agendado com sucesso.');

    api.createError = new HttpErrorResponse({ status: 409, statusText: 'Conflict' });
    component.openCreateDialog();
    component.createForm = { workoutId: 'workout-1', scheduledDate: '2026-08-24' };
    await component.createEntry();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Este treino já está agendado para esta data.');
  });

  it('prevents scheduling when there are no active workouts', async () => {
    const { fixture, component } = await renderPage(adminUser, [], [workout({ active: false })]);

    component.openCreateDialog();
    fixture.detectChanges();

    expect(component.createDialogOpen).toBe(false);
    expect(fixture.nativeElement.textContent).toContain('Cadastre ou ative pelo menos um treino antes de agendar.');
  });

  it('edits entry while preserving an existing inactive workout option', async () => {
    const inactiveEntry = entry({
      id: 'entry-1',
      workout: { id: 'inactive-workout', name: 'Treino Antigo', active: false },
    });
    const api = new FakeScheduleApi([inactiveEntry]);
    const { component } = await renderPage(adminUser, [inactiveEntry], [workout({ id: 'active-workout', name: 'Treino Novo' })], api);

    component.openEditDialog(inactiveEntry);

    expect(component.availableWorkouts(inactiveEntry.workout).map((item: WorkoutListItem) => item.id)).toEqual(['inactive-workout', 'active-workout']);
    expect(component.workoutOptionLabel(inactiveEntry.workout)).toBe('Treino Antigo (inativo)');

    await component.updateEntry();
    expect(api.updated[0]).toEqual({ id: 'entry-1', request: { workoutId: 'inactive-workout', scheduledDate: '2026-08-24' } });
  });

  it('deletes entry after explicit confirmation', async () => {
    const original = entry({ id: 'entry-1', workout: { id: 'workout-1', name: 'Treino A', active: true } });
    const api = new FakeScheduleApi([original]);
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], api);

    component.confirmDelete(original);
    expect(component.confirmationTitle()).toBe('Remover Treino A da agenda de 24/08/2026?');
    await component.applyConfirmation();
    fixture.detectChanges();

    expect(api.deleted).toEqual(['entry-1']);
    expect(fixture.nativeElement.textContent).toContain('Nenhum treino agendado neste período.');
  });

  it('completes entry after confirmation for ADMIN and PROFESSOR', async () => {
    const original = entry({ id: 'entry-1', workout: { id: 'workout-1', name: 'Treino A', active: true } });
    const adminApi = new FakeScheduleApi([original]);
    const admin = await renderPage(adminUser, [original], [workout()], adminApi);

    admin.component.confirmComplete(original);
    admin.fixture.detectChanges();
    expect(admin.component.completeDialogOpen()).toBe(true);
    expect(admin.fixture.nativeElement.textContent).toContain('Registrar treino realizado');
    expect(admin.fixture.nativeElement.textContent).toContain('Treino A');
    expect(admin.fixture.nativeElement.textContent).toContain('24/08/2026');
    expect(admin.fixture.nativeElement.textContent).toContain('Após registrar o treino como realizado');
    await admin.component.completeEntry();

    expect(adminApi.completed).toEqual([{ id: 'entry-1', details: {} }]);

    const professorApi = new FakeScheduleApi([original]);
    const professor = await renderPage(professorUser, [original], [workout()], professorApi);
    professor.component.confirmComplete(original);
    await professor.component.completeEntry();

    expect(professorApi.completed).toEqual([{ id: 'entry-1', details: {} }]);
  });

  it('submits selected students as participant ids', async () => {
    const original = entry({ id: 'entry-1' });
    const api = new FakeScheduleApi([original]);
    const joao = student({ id: 'student-1', name: 'João' });
    const maria = student({ id: 'student-2', name: 'Maria' });
    const studentApi = new FakeStudentApi([joao, maria]);
    const { component } = await renderPage(adminUser, [original], [workout()], api, undefined, studentApi);

    component.confirmComplete(original);
    await component.loadCompletionStudents();
    await component.completeEntry();
    expect(api.completed.at(-1)).toEqual({ id: 'entry-1', details: {} });

    component.confirmComplete(original);
    await component.loadCompletionStudents();
    component.selectStudent(joao);
    component.selectStudent(maria);
    await component.completeEntry();
    expect(api.completed.at(-1)).toEqual({ id: 'entry-1', details: { participantStudentIds: ['student-1', 'student-2'] } });
  });

  it('shows loading state while fetching students for the completion dialog', async () => {
    const original = entry({ id: 'entry-1' });
    const studentApi = new FakeStudentApi([student()]);
    studentApi.listGate = pendingPromise();
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], undefined, undefined, studentApi);

    component.completionEntry = original;
    component.completeDialogOpen.set(true);
    const loading = component.loadCompletionStudents();
    fixture.detectChanges();

    expect(component.completionStudentsLoading()).toBe(true);
    expect(fixture.nativeElement.textContent).toContain('Carregando alunos');

    studentApi.listGate.resolve();
    await loading;
    fixture.detectChanges();
    expect(component.completionStudentsLoading()).toBe(false);
  });

  it('shows an error state when students fail to load for the completion dialog', async () => {
    const original = entry({ id: 'entry-1' });
    const studentApi = new FakeStudentApi([]);
    studentApi.listError = new HttpErrorResponse({ status: 500, statusText: 'Server Error' });
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], undefined, undefined, studentApi);

    component.confirmComplete(original);
    await component.loadCompletionStudents();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Não foi possível carregar os alunos. Tente novamente.');
  });

  it('shows an empty state when there are no active students for the completion dialog', async () => {
    const original = entry({ id: 'entry-1' });
    const studentApi = new FakeStudentApi([student({ active: false })]);
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], undefined, undefined, studentApi);

    component.confirmComplete(original);
    await component.loadCompletionStudents();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Nenhum aluno ativo cadastrado');
  });

  it('searches active students, prevents duplicate selection and validates completion form', async () => {
    const original = entry({ id: 'entry-1' });
    const joao = student({ id: 'student-1', name: 'João Silva' });
    const inactiveStudent = student({ id: 'student-2', name: 'Maria Inativa', active: false });
    const studentApi = new FakeStudentApi([joao, inactiveStudent]);
    const { component } = await renderPage(adminUser, [original], [workout()], undefined, undefined, studentApi);

    component.confirmComplete(original);
    await component.loadCompletionStudents();

    expect(component.filteredCompletionStudents()).toEqual([joao]);

    component.completionForm.studentSearch = 'joão';
    expect(component.filteredCompletionStudents()).toEqual([joao]);

    component.completionForm.studentSearch = 'inativa';
    expect(component.filteredCompletionStudents()).toEqual([]);

    component.completionForm.studentSearch = '';
    component.selectStudent(joao);
    expect(component.completionForm.selectedStudents).toEqual([joao]);
    expect(component.filteredCompletionStudents()).toEqual([]);

    component.selectStudent(joao);
    expect(component.completionForm.selectedStudents).toEqual([joao]);

    component.removeSelectedStudent(0);
    expect(component.completionForm.selectedStudents).toEqual([]);

    expect(component.completionFormValid()).toBe(true);
    component.completionForm.notes = 'a'.repeat(2001);
    expect(component.completionFormValid()).toBe(false);
  });

  it('shows selected students in the dialog and keeps selection order', async () => {
    const original = entry({ id: 'entry-1' });
    const api = new FakeScheduleApi([original]);
    const joao = student({ id: 'student-1', name: 'Alexandre Denicol' });
    const alan = student({ id: 'student-2', name: 'Alan Fogaça' });
    const studentApi = new FakeStudentApi([joao, alan]);
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], api, undefined, studentApi);

    component.confirmComplete(original);
    await component.loadCompletionStudents();
    fixture.detectChanges();

    setInputValue(fixture.nativeElement.querySelector('input[name="student-search"]'), 'Alexandre');
    fixture.detectChanges();
    clickButton(fixture.nativeElement, 'Alexandre Denicol');
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Participantes selecionados (1)');
    expect(fixture.nativeElement.textContent).toContain('Alexandre Denicol');
    expect((fixture.nativeElement.querySelector('input[name="student-search"]') as HTMLInputElement).value).toBe('');
    expect(api.completed).toHaveLength(0);

    clickButton(fixture.nativeElement, 'Alan Fogaça');
    fixture.detectChanges();

    const text = fixture.nativeElement.textContent;
    expect(text).toContain('Participantes selecionados (2)');
    expect(text.indexOf('Alexandre Denicol')).toBeLessThan(text.indexOf('Alan Fogaça'));
    expect(component.completionForm.selectedStudents.map((s: Student) => s.id)).toEqual(['student-1', 'student-2']);
  });

  it('removes only the selected participant without completing training', async () => {
    const original = entry({ id: 'entry-1' });
    const api = new FakeScheduleApi([original]);
    const joao = student({ id: 'student-1', name: 'Alexandre Denicol' });
    const alan = student({ id: 'student-2', name: 'Alan Fogaça' });
    const maria = student({ id: 'student-3', name: 'Maria' });
    const studentApi = new FakeStudentApi([joao, alan, maria]);
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], api, undefined, studentApi);

    component.confirmComplete(original);
    await component.loadCompletionStudents();
    component.completionForm.selectedStudents = [joao, alan, maria];
    fixture.detectChanges();

    const removeAlan = fixture.nativeElement.querySelector('button[aria-label="Remover participante Alan Fogaça"]') as HTMLButtonElement;
    removeAlan.click();
    fixture.detectChanges();

    expect(component.completionForm.selectedStudents.map((s: Student) => s.id)).toEqual(['student-1', 'student-3']);
    const selectedListText = fixture.nativeElement.querySelector('.participant-list')?.textContent ?? '';
    expect(selectedListText).toContain('Alexandre Denicol');
    expect(selectedListText).toContain('Maria');
    expect(selectedListText).not.toContain('Alan Fogaça');
    expect(api.completed).toHaveLength(0);
  });

  it('final completion button submits exactly once with participant details and refreshes agenda', async () => {
    const original = entry({ id: 'entry-1' });
    const api = new FakeScheduleApi([original]);
    const joao = student({ id: 'student-1', name: 'Alexandre Denicol' });
    const alan = student({ id: 'student-2', name: 'Alan Fogaça' });
    const studentApi = new FakeStudentApi([joao, alan]);
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], api, undefined, studentApi);

    component.confirmComplete(original);
    await component.loadCompletionStudents();
    component.completionForm.selectedStudents = [joao, alan];
    component.completionForm.notes = '  Boa resposta da turma.\nLinha 2  ';
    const previousLoadCount = api.ranges.length;
    fixture.detectChanges();

    clickButton(fixture.nativeElement, 'Registrar como realizado');
    await fixture.whenStable();
    fixture.detectChanges();

    expect(api.completed).toEqual([
      {
        id: 'entry-1',
        details: {
          participantStudentIds: ['student-1', 'student-2'],
          notes: 'Boa resposta da turma.\nLinha 2',
        },
      },
    ]);
    expect(api.ranges.length).toBeGreaterThan(previousLoadCount);
    expect(component.completeDialogOpen()).toBe(false);
    expect(component.completionForm).toEqual({ studentSearch: '', selectedStudents: [], notes: '' });
    expect(fixture.nativeElement.textContent).toContain('Treino registrado como realizado.');
  });

  it('empty details completion omits missing count and notes instead of sending canonical empty values', async () => {
    const original = entry({ id: 'entry-1' });
    const api = new FakeScheduleApi([original]);
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], api);

    component.confirmComplete(original);
    component.completionForm.notes = '   \n  ';
    fixture.detectChanges();
    clickButton(fixture.nativeElement, 'Registrar como realizado');
    await fixture.whenStable();

    expect(api.completed[0]).toEqual({ id: 'entry-1', details: {} });
  });

  it('shows optional completion notes textarea and character counter', async () => {
    const original = entry({ id: 'entry-1' });
    const api = new FakeScheduleApi([original]);
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], api);

    component.confirmComplete(original);
    fixture.detectChanges();

    const textarea = fixture.nativeElement.querySelector('textarea[name="completion-notes"]') as HTMLTextAreaElement;
    expect(fixture.nativeElement.textContent).toContain('Observações (opcional)');
    expect(textarea).toBeTruthy();
    expect(textarea.getAttribute('maxlength')).toBe('2000');
    expect(textarea.getAttribute('placeholder')).toBe('Ex.: adaptações realizadas, dificuldades ou observações gerais do treino.');
    expect(fixture.nativeElement.textContent).toContain('0 / 2000');

    setInputValue(textarea, 'Linha 1\nLinha 2');
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('15 / 2000');
  });

  it('rejects completion notes above the frontend limit without submitting', async () => {
    const original = entry({ id: 'entry-1' });
    const api = new FakeScheduleApi([original]);
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], api);

    component.confirmComplete(original);
    component.completionForm.notes = 'a'.repeat(2001);
    fixture.detectChanges();

    const submit = Array.from(fixture.nativeElement.querySelectorAll('button') as NodeListOf<HTMLButtonElement>)
      .find((button) => button.textContent?.includes('Registrar como realizado'));
    if (!submit) {
      throw new Error('completion submit button not found');
    }
    expect(component.completionFormValid()).toBe(false);
    expect(submit.disabled).toBe(true);

    await component.completeEntry();
    expect(api.completed).toHaveLength(0);
  });

  it('clears notes after successful completion and does not reopen with previous text', async () => {
    const original = entry({ id: 'entry-1' });
    const api = new FakeScheduleApi([original]);
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], api);

    component.confirmComplete(original);
    component.completionForm.notes = 'Observação temporária';
    await component.completeEntry();
    fixture.detectChanges();

    expect(component.completionForm.notes).toBe('');

    component.confirmComplete(original);
    fixture.detectChanges();
    const textarea = fixture.nativeElement.querySelector('textarea[name="completion-notes"]') as HTMLTextAreaElement;
    expect(textarea.value).toBe('');
  });

  it('prevents duplicate completion while request is pending', async () => {
    const original = entry({ id: 'entry-1' });
    const api = new FakeScheduleApi([original]);
    api.completeGate = pendingPromise();
    const { component } = await renderPage(adminUser, [original], [workout()], api);

    component.confirmComplete(original);
    const first = component.completeEntry();
    const second = component.completeEntry();

    expect(api.completed).toHaveLength(1);
    api.completeGate.resolve();
    await first;
    await second;
    expect(api.completed).toHaveLength(1);
  });

  it('failed completion keeps dialog open and re-enables submit', async () => {
    const original = entry({ id: 'entry-1' });
    const api = new FakeScheduleApi([original]);
    api.completeError = new HttpErrorResponse({ status: 500, statusText: 'Server Error' });
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], api);

    component.confirmComplete(original);
    await component.completeEntry();
    fixture.detectChanges();

    expect(component.completeDialogOpen()).toBe(true);
    expect(component.submitting()).toBe(false);
    expect(fixture.nativeElement.textContent).toContain('Não foi possível registrar o treino como realizado. Tente novamente.');
  });

  it('shows completed state and hides impossible actions', async () => {
    const completed = entry({ completedAt: '2026-08-23T15:32:00Z' });
    const { fixture } = await renderPage(adminUser, [completed], [workout()]);
    const text = fixture.nativeElement.textContent;

    expect(text).toContain('Realizado');
    expect(text).not.toContain('Ver treino');
    expect(text).not.toContain('Marcar como realizado');
    expect(text).not.toContain('Editar');
    expect(text).not.toContain('Remover');
  });

  it('shows duplicate completion and completed immutability errors', async () => {
    const original = entry({ id: 'entry-1' });
    const api = new FakeScheduleApi([original]);
    api.completeError = new HttpErrorResponse({ status: 409, statusText: 'Conflict' });
    const { fixture, component } = await renderPage(adminUser, [original], [workout()], api);

    component.confirmComplete(original);
    await component.completeEntry();
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Este treino já foi registrado como realizado.');

    api.updateError = new HttpErrorResponse({ status: 409, statusText: 'Conflict' });
    component.openEditDialog(original);
    await component.updateEntry();
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Treinos já realizados não podem ser alterados ou removidos.');

    api.deleteError = new HttpErrorResponse({ status: 409, statusText: 'Conflict' });
    component.confirmDelete(original);
    await component.applyConfirmation();
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Treinos já realizados não podem ser alterados ou removidos.');
  });

  it('clears form state after dialog closes', async () => {
    const original = entry();
    const { component } = await renderPage(adminUser, [original], [workout()]);

    component.openCreateDialog();
    component.createForm = { workoutId: 'workout-id', scheduledDate: '2026-08-24' };
    component.closeCreateDialog();
    expect(component.createForm).toEqual({ scheduledDate: '', workoutId: '' });

    component.openEditDialog(original);
    component.editForm = { workoutId: 'workout-id', scheduledDate: '2026-08-25' };
    component.closeEditDialog();
    expect(component.editForm).toEqual({ scheduledDate: '', workoutId: '' });

    component.confirmComplete(original);
    component.completionForm = { studentSearch: 'Mar', selectedStudents: [student({ id: 'student-1', name: 'João' })], notes: 'Observação temporária' };
    component.closeCompleteDialog();
    expect(component.completionForm).toEqual({ studentSearch: '', selectedStudents: [], notes: '' });
  });
});

async function renderPage(
  user: AuthUser,
  entries: ScheduleEntry[],
  workouts: WorkoutListItem[],
  api = new FakeScheduleApi(entries),
  workoutApi = new FakeWorkoutApi(workouts),
  studentApi = new FakeStudentApi([]),
) {
  TestBed.resetTestingModule();

  const currentUser = signal<AuthUser | null>(user);
  await TestBed.configureTestingModule({
    imports: [SchedulePage],
    providers: [
      provideRouter([]),
      { provide: ScheduleApiService, useValue: api },
      { provide: WorkoutApiService, useValue: workoutApi },
      { provide: StudentApiService, useValue: studentApi },
      {
        provide: AuthService,
        useValue: {
          currentUser: currentUser.asReadonly(),
        },
      },
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(SchedulePage);
  const component = fixture.componentInstance as any;
  fixture.detectChanges();
  await fixture.whenStable();
  await component.loadData();
  fixture.detectChanges();

  return { fixture, component, api, workoutApi, studentApi };
}

function entry(overrides: Partial<ScheduleEntry> = {}): ScheduleEntry {
  return {
    id: 'entry-id',
    scheduledDate: '2026-08-24',
    workout: { id: 'workout-id', name: 'Treino Base', active: true },
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

function student(overrides: Partial<Student> = {}): Student {
  return {
    id: 'student-id',
    name: 'Aluno Padrão',
    active: true,
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

class FakeStudentApi {
  listCalls = 0;
  listError: unknown = null;
  listGate: DeferredPromise | null = null;

  constructor(private students: Student[]) {}

  async list(): Promise<Student[]> {
    this.listCalls += 1;
    if (this.listGate) {
      await this.listGate.promise;
    }
    if (this.listError) {
      throw this.listError;
    }
    return this.students;
  }
}

function workout(overrides: Partial<WorkoutListItem> = {}): WorkoutListItem {
  return {
    id: 'workout-id',
    name: 'Treino Base',
    active: true,
    blockCount: 2,
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

class FakeScheduleApi {
  ranges: Array<{ from: string; to: string }> = [];
  created: Array<{ workoutId: string; scheduledDate: string }> = [];
  updated: Array<{ id: string; request: { workoutId: string; scheduledDate: string } }> = [];
  deleted: string[] = [];
  completed: Array<{ id: string; details: unknown }> = [];
  createError: unknown = null;
  updateError: unknown = null;
  deleteError: unknown = null;
  completeError: unknown = null;
  completeGate: DeferredPromise | null = null;

  constructor(private entries: ScheduleEntry[]) {}

  async list(from: string, to: string): Promise<ScheduleEntry[]> {
    this.ranges.push({ from, to });
    return this.entries;
  }

  async create(request: { workoutId: string; scheduledDate: string }): Promise<ScheduleEntry> {
    this.created.push(request);
    if (this.createError) {
      throw this.createError;
    }
    const created = entry({
      id: `entry-${this.created.length}`,
      scheduledDate: request.scheduledDate,
      workout: { id: request.workoutId, name: 'Treino Base', active: true },
    });
    this.entries = [...this.entries, created];
    return created;
  }

  async update(id: string, request: { workoutId: string; scheduledDate: string }): Promise<ScheduleEntry> {
    this.updated.push({ id, request });
    if (this.updateError) {
      throw this.updateError;
    }
    const current = this.entries.find((item) => item.id === id) ?? entry({ id });
    const updated = {
      ...current,
      scheduledDate: request.scheduledDate,
      workout: { ...current.workout, id: request.workoutId },
    };
    this.entries = this.entries.map((item) => item.id === id ? updated : item);
    return updated;
  }

  async delete(id: string): Promise<void> {
    this.deleted.push(id);
    if (this.deleteError) {
      throw this.deleteError;
    }
    this.entries = this.entries.filter((item) => item.id !== id);
  }

  async complete(id: string, details: unknown = {}): Promise<unknown> {
    this.completed.push({ id, details });
    if (this.completeError) {
      throw this.completeError;
    }
    if (this.completeGate) {
      await this.completeGate.promise;
    }
    this.entries = this.entries.map((item) => item.id === id ? { ...item, completedAt: '2026-08-23T15:32:00Z' } : item);
    return {};
  }
}

interface DeferredPromise {
  promise: Promise<void>;
  resolve: () => void;
}

function pendingPromise(): DeferredPromise {
  let resolve: () => void = () => undefined;
  const promise = new Promise<void>((resolver) => {
    resolve = resolver;
  });

  return { promise, resolve };
}

function setInputValue(input: HTMLInputElement | HTMLTextAreaElement | null, value: string): void {
  if (!input) {
    throw new Error('input not found');
  }

  input.value = value;
  input.dispatchEvent(new Event('input', { bubbles: true }));
}

function clickButton(root: HTMLElement, text: string): void {
  const button = Array.from(root.querySelectorAll('button')).find((candidate) => candidate.textContent?.includes(text));
  if (!button) {
    throw new Error(`button not found: ${text}`);
  }

  (button as HTMLButtonElement).click();
}

class FakeWorkoutApi {
  listCalls = 0;
  loadedDetails: string[] = [];
  details = new Map<string, WorkoutDetail>();
  detailError: unknown = null;
  detailGate: DeferredPromise | null = null;

  constructor(private workouts: WorkoutListItem[]) {}

  async list(): Promise<WorkoutListItem[]> {
    this.listCalls += 1;
    return this.workouts;
  }

  async getById(id: string): Promise<WorkoutDetail> {
    this.loadedDetails.push(id);
    if (this.detailError) {
      throw this.detailError;
    }
    if (this.detailGate) {
      await this.detailGate.promise;
    }
    return this.details.get(id) ?? workoutDetail({ id });
  }
}

function workoutDetail(overrides: Partial<WorkoutDetail> = {}): WorkoutDetail {
  return {
    id: 'workout-id',
    name: 'Treino Base',
    active: true,
    blocks: [workoutBlock()],
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

function workoutBlock(overrides: Partial<WorkoutDetail['blocks'][number]> = {}): WorkoutDetail['blocks'][number] {
  return {
    id: 'block-id',
    name: 'Bloco Base',
    description: null,
    sequence: [],
    active: true,
    position: 1,
    category: { id: 'category-id', name: 'Categoria Base' },
    ...overrides,
  };
}

function detailBodyText(root: HTMLElement): string {
  const body = root.querySelector('.workout-detail-body');
  if (!body) {
    throw new Error('workout detail body not found');
  }

  return body.textContent ?? '';
}
