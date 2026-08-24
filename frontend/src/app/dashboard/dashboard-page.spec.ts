import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { AuthUser } from '../auth/auth.models';
import { AuthService } from '../auth/auth.service';
import { Block, BlockApiService } from '../blocks/block-api.service';
import { Category, CategoryApiService } from '../categories/category-api.service';
import { HistoryApiService, HistoryDetail, HistoryListItem } from '../history/history-api.service';
import { ScheduleApiService, ScheduleEntry } from '../schedule/schedule-api.service';
import { WorkoutApiService, WorkoutDetail, WorkoutListItem } from '../workouts/workout-api.service';
import { DashboardPage } from './dashboard-page';

const adminUser: AuthUser = {
  id: 'admin-id',
  name: 'Alexandre',
  email: 'admin@example.com',
  role: 'ADMIN',
};

const professorUser: AuthUser = {
  id: 'professor-id',
  name: 'Professor',
  email: 'professor@example.com',
  role: 'PROFESSOR',
};

describe('DashboardPage', () => {
  it('renders real summary data for ADMIN', async () => {
    const { fixture } = await renderDashboard(adminUser, {
      schedule: [scheduleEntry(), scheduleEntry({ id: 'completed', completedAt: '2026-08-23T12:00:00Z' })],
      workouts: [workout(), workout({ id: 'workout-2' })],
      blocks: [block(), block({ id: 'block-2' }), block({ id: 'block-3' })],
      categories: [category(), category({ id: 'category-2' })],
    });
    const text = fixture.nativeElement.textContent;

    expect(text).toContain('Olá, Alexandre');
    expect(text).toContain('Treinos agendados');
    expect(text).toContain('Total de treinos');
    expect(text).toContain('Total de blocos');
    expect(text).toContain('Total de categorias');
    expect(text).toContain('1');
    expect(text).toContain('2');
    expect(text).toContain('3');
  });

  it('renders for PROFESSOR with operational dashboard access', async () => {
    const { fixture } = await renderDashboard(professorUser, {
      schedule: [scheduleEntry({ workout: { id: 'workout-1', name: 'Treino Técnico', active: true } })],
    });

    expect(fixture.nativeElement.textContent).toContain('Olá, Professor');
    expect(fixture.nativeElement.textContent).toContain('Treino Técnico');
    expect(fixture.nativeElement.textContent).not.toContain('Professores');
  });

  it('shows intentional empty states', async () => {
    const { fixture } = await renderDashboard(adminUser);
    const text = fixture.nativeElement.textContent;

    expect(text).toContain('Nenhum treino agendado.');
    expect(text).toContain('Nenhum treino realizado neste período.');
    expect(text).toContain('Abrir agenda');
    expect(text).toContain('Abrir histórico');
  });

  it('provides navigation actions from summary and sections', async () => {
    const { fixture } = await renderDashboard(adminUser);
    const links = Array.from(fixture.nativeElement.querySelectorAll('a') as NodeListOf<HTMLAnchorElement>)
      .map((link) => link.getAttribute('href'));

    expect(links).toContain('/app/agenda');
    expect(links).toContain('/app/treinos');
    expect(links).toContain('/app/blocos');
    expect(links).toContain('/app/categorias');
    expect(links).toContain('/app/historico');
  });

  it('shows upcoming trainings from real agenda data and opens scheduled workout detail', async () => {
    const workoutApi = new FakeWorkoutApi([workout()]);
    workoutApi.details.set('workout-1', workoutDetail({
      name: 'Treino Técnico',
      blocks: [
        workoutBlock({ id: 'block-1', name: 'Aquecimento', position: 1, category: { id: 'category-1', name: 'Preparação' } }),
        workoutBlock({ id: 'block-2', name: 'Jab + direto', position: 2, category: { id: 'category-2', name: 'Técnica' } }),
      ],
    }));
    const upcoming = scheduleEntry({
      workout: { id: 'workout-1', name: 'Treino Técnico', active: true },
      scheduledDate: '2026-08-24',
    });
    const { fixture, component } = await renderDashboard(adminUser, { schedule: [upcoming], workoutApi });

    expect(fixture.nativeElement.textContent).toContain('Próximos treinos');
    expect(fixture.nativeElement.textContent).toContain('Treino Técnico');

    await component.openWorkoutDetail(upcoming);
    fixture.detectChanges();

    expect(workoutApi.loadedDetails).toEqual(['workout-1']);
    expect(fixture.nativeElement.textContent).toContain('Aquecimento');
    expect(fixture.nativeElement.textContent).toContain('Preparação');
    expect(fixture.nativeElement.textContent).toContain('Jab + direto');
  });

  it('shows recent immutable history and opens history detail', async () => {
    const historyApi = new FakeHistoryApi([
      historyItem({ id: 'history-1', workoutName: 'Treino Realizado', participantCount: 12 }),
    ]);
    historyApi.details.set('history-1', historyDetail({
      id: 'history-1',
      workoutName: 'Treino Realizado',
      participantCount: 12,
      notes: 'Boa resposta da turma.',
    }));
    const { fixture, component } = await renderDashboard(adminUser, { historyApi });
    const item = component.recentHistory()[0] as HistoryListItem;

    expect(fixture.nativeElement.textContent).toContain('Últimos treinos realizados');
    expect(fixture.nativeElement.textContent).toContain('Treino Realizado');
    expect(fixture.nativeElement.textContent).toContain('12 participantes');

    await component.openHistoryDetail(item);
    fixture.detectChanges();

    expect(historyApi.loadedDetails).toEqual(['history-1']);
    expect(fixture.nativeElement.textContent).toContain('Boa resposta da turma.');
  });
});

interface DashboardFixtures {
  schedule?: ScheduleEntry[];
  workouts?: WorkoutListItem[];
  blocks?: Block[];
  categories?: Category[];
  history?: HistoryListItem[];
  workoutApi?: FakeWorkoutApi;
  historyApi?: FakeHistoryApi;
}

async function renderDashboard(user: AuthUser, fixtures: DashboardFixtures = {}) {
  TestBed.resetTestingModule();

  const currentUser = signal<AuthUser | null>(user);
  const workoutApi = fixtures.workoutApi ?? new FakeWorkoutApi(fixtures.workouts ?? []);
  const historyApi = fixtures.historyApi ?? new FakeHistoryApi(fixtures.history ?? []);

  await TestBed.configureTestingModule({
    imports: [DashboardPage],
    providers: [
      provideRouter([]),
      { provide: ScheduleApiService, useValue: new FakeScheduleApi(fixtures.schedule ?? []) },
      { provide: WorkoutApiService, useValue: workoutApi },
      { provide: BlockApiService, useValue: new FakeBlockApi(fixtures.blocks ?? []) },
      { provide: CategoryApiService, useValue: new FakeCategoryApi(fixtures.categories ?? []) },
      { provide: HistoryApiService, useValue: historyApi },
      {
        provide: AuthService,
        useValue: {
          currentUser: currentUser.asReadonly(),
        },
      },
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(DashboardPage);
  const component = fixture.componentInstance as any;
  fixture.detectChanges();
  await fixture.whenStable();
  await component.loadDashboard();
  fixture.detectChanges();

  return { fixture, component, workoutApi, historyApi };
}

function scheduleEntry(overrides: Partial<ScheduleEntry> = {}): ScheduleEntry {
  return {
    id: 'entry-1',
    scheduledDate: '2026-08-24',
    workout: { id: 'workout-1', name: 'Treino Base', active: true },
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

function workout(overrides: Partial<WorkoutListItem> = {}): WorkoutListItem {
  return {
    id: 'workout-1',
    name: 'Treino Base',
    active: true,
    blockCount: 1,
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

function workoutDetail(overrides: Partial<WorkoutDetail> = {}): WorkoutDetail {
  return {
    id: 'workout-1',
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
    id: 'block-1',
    name: 'Bloco',
    description: null,
    sequence: [],
    active: true,
    position: 1,
    category: { id: 'category-1', name: 'Categoria' },
    ...overrides,
  };
}

function block(overrides: Partial<Block> = {}): Block {
  return {
    id: 'block-1',
    name: 'Bloco',
    description: null,
    sequence: [],
    active: true,
    category: { id: 'category-1', name: 'Categoria' },
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

function category(overrides: Partial<Category> = {}): Category {
  return {
    id: 'category-1',
    name: 'Categoria',
    active: true,
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

function historyItem(overrides: Partial<HistoryListItem> = {}): HistoryListItem {
  return {
    id: 'history-1',
    trainingDate: '2026-08-20',
    workoutName: 'Treino Realizado',
    blockCount: 2,
    participantCount: null,
    completedByName: 'Professor',
    completedAt: '2026-08-20T12:00:00Z',
    scheduleEntryId: 'entry-1',
    ...overrides,
  };
}

function historyDetail(overrides: Partial<HistoryDetail> = {}): HistoryDetail {
  return {
    id: 'history-1',
    trainingDate: '2026-08-20',
    workoutName: 'Treino Realizado',
    participantCount: null,
    participantNames: [],
    notes: null,
    completedByName: 'Professor',
    completedAt: '2026-08-20T12:00:00Z',
    blocks: [{ position: 1, blockName: 'Bloco', categoryName: 'Categoria', description: null, sequence: [] }],
    ...overrides,
  };
}

class FakeScheduleApi {
  ranges: Array<{ from: string; to: string }> = [];

  constructor(private entries: ScheduleEntry[]) {}

  async list(from: string, to: string): Promise<ScheduleEntry[]> {
    this.ranges.push({ from, to });
    return this.entries;
  }
}

class FakeWorkoutApi {
  loadedDetails: string[] = [];
  details = new Map<string, WorkoutDetail>();

  constructor(private workouts: WorkoutListItem[]) {}

  async list(): Promise<WorkoutListItem[]> {
    return this.workouts;
  }

  async getById(id: string): Promise<WorkoutDetail> {
    this.loadedDetails.push(id);
    return this.details.get(id) ?? workoutDetail({ id });
  }
}

class FakeBlockApi {
  constructor(private blocks: Block[]) {}

  async list(): Promise<Block[]> {
    return this.blocks;
  }
}

class FakeCategoryApi {
  constructor(private categories: Category[]) {}

  async list(): Promise<Category[]> {
    return this.categories;
  }
}

class FakeHistoryApi {
  loadedDetails: string[] = [];
  details = new Map<string, HistoryDetail>();

  constructor(private items: HistoryListItem[]) {}

  async list(): Promise<HistoryListItem[]> {
    return this.items;
  }

  async getById(id: string): Promise<HistoryDetail> {
    this.loadedDetails.push(id);
    return this.details.get(id) ?? historyDetail({ id });
  }
}
