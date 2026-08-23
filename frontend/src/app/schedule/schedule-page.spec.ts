import { HttpErrorResponse } from '@angular/common/http';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { AuthUser } from '../auth/auth.models';
import { AuthService } from '../auth/auth.service';
import { WorkoutApiService, WorkoutListItem } from '../workouts/workout-api.service';
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

  it('allows PROFESSOR read-only access without mutation controls or workout option loading', async () => {
    const workoutApi = new FakeWorkoutApi([workout()]);
    const { fixture } = await renderPage(professorUser, [entry()], [workout()], undefined, workoutApi);
    const text = fixture.nativeElement.textContent;

    expect(text).toContain('Treino Base');
    expect(text).not.toContain('Agendar treino');
    expect(text).not.toContain('Editar');
    expect(text).not.toContain('Remover');
    expect(workoutApi.listCalls).toBe(0);
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
  });
});

async function renderPage(
  user: AuthUser,
  entries: ScheduleEntry[],
  workouts: WorkoutListItem[],
  api = new FakeScheduleApi(entries),
  workoutApi = new FakeWorkoutApi(workouts),
) {
  TestBed.resetTestingModule();

  const currentUser = signal<AuthUser | null>(user);
  await TestBed.configureTestingModule({
    imports: [SchedulePage],
    providers: [
      provideRouter([]),
      { provide: ScheduleApiService, useValue: api },
      { provide: WorkoutApiService, useValue: workoutApi },
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

  return { fixture, component, api, workoutApi };
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
  createError: unknown = null;

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
    this.entries = this.entries.filter((item) => item.id !== id);
  }
}

class FakeWorkoutApi {
  listCalls = 0;

  constructor(private workouts: WorkoutListItem[]) {}

  async list(): Promise<WorkoutListItem[]> {
    this.listCalls += 1;
    return this.workouts;
  }
}
