import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { HistoryApiService, HistoryDetail, HistoryListItem } from './history-api.service';
import { HistoryPage } from './history-page';

describe('HistoryPage', () => {
  it('shows empty state', async () => {
    const { fixture } = await renderPage([]);

    expect(fixture.nativeElement.textContent).toContain('Nenhum treino realizado neste período.');
  });

  it('shows populated history grouped newest first with snapshot values', async () => {
    const items = [
      historyItem({ id: 'older', trainingDate: '2026-08-20', workoutName: 'Nome Antigo', completedByName: 'Professor Snapshot', blockCount: 4, participantCount: 2 }),
      historyItem({ id: 'newer', trainingDate: '2026-08-24', workoutName: 'Treino Snapshot', completedByName: 'Alexandre', blockCount: 1, participantCount: 1 }),
    ];
    const { fixture } = await renderPage(items);
    const text = fixture.nativeElement.textContent;

    expect(text).toContain('Histórico');
    expect(text).toContain('Consulte os treinos realizados.');
    expect(text).toContain('24/08/2026');
    expect(text).toContain('20/08/2026');
    expect(text).toContain('Treino Snapshot');
    expect(text).toContain('Realizado por Alexandre');
    expect(text).toContain('4 blocos');
    expect(text).toContain('1 aluno');
    expect(text).toContain('2 alunos');
    expect(text.indexOf('Treino Snapshot')).toBeLessThan(text.indexOf('Nome Antigo'));
  });

  it('does not invent zero participants when participant count is null', async () => {
    const { fixture } = await renderPage([historyItem({ participantCount: null })]);

    expect(fixture.nativeElement.textContent).not.toContain('0 alunos');
  });

  it('formats DATE values without timezone shifting and formats completion timestamp', async () => {
    const { component } = await renderPage([historyItem({ trainingDate: '2026-08-01' })]);

    expect(component.formatDate('2026-08-01')).toBe('01/08/2026');
    expect(component.formatCompletedAt('2026-08-24T15:32:00Z')).toContain('2026');
    expect(component.formatCompletedTime('2026-08-24T15:32:00Z')).not.toBe('-');
  });

  it('requests bounded month ranges for navigation', async () => {
    const api = new FakeHistoryApi([]);
    const { component } = await renderPage([], api);

    component.currentMonth.set(new Date(2026, 7, 1));
    await component.loadHistory();
    expect(api.ranges.at(-1)).toEqual({ from: '2026-08-01', to: '2026-08-31' });

    await component.nextMonth();
    expect(api.ranges.at(-1)).toEqual({ from: '2026-09-01', to: '2026-09-30' });

    await component.previousMonth();
    expect(api.ranges.at(-1)).toEqual({ from: '2026-08-01', to: '2026-08-31' });

    await component.goToToday();
    expect(api.ranges.at(-1)?.from).toMatch(/^\d{4}-\d{2}-01$/);
  });

  it('opens detail with ordered snapshot blocks and category names', async () => {
    const api = new FakeHistoryApi([historyItem({ id: 'history-1' })]);
    api.details.set('history-1', historyDetail({
      id: 'history-1',
      blocks: [
        { position: 1, blockName: 'Bloco Snapshot A', categoryName: 'Categoria Snapshot A' },
        { position: 2, blockName: 'Bloco Snapshot B', categoryName: 'Categoria Snapshot B' },
      ],
      participantCount: 12,
      participantNames: ['João', 'Maria', 'Pedro'],
    }));
    const { fixture, component } = await renderPage(api.items, api);

    await component.openDetail(api.items[0]);
    fixture.detectChanges();
    const text = fixture.nativeElement.textContent;

    expect(api.loadedDetails).toEqual(['history-1']);
    expect(text).toContain('Detalhes do histórico');
    expect(text).toContain('Bloco Snapshot A');
    expect(text).toContain('Categoria Snapshot A');
    expect(text).toContain('Participação');
    expect(text).toContain('Quantidade: 12 alunos');
    expect(text).toContain('João');
    expect(text.indexOf('João')).toBeLessThan(text.indexOf('Maria'));
    expect(text.indexOf('Bloco Snapshot A')).toBeLessThan(text.indexOf('Bloco Snapshot B'));
  });

  it('renders count-only, names-only and missing participation detail states', async () => {
    const api = new FakeHistoryApi([
      historyItem({ id: 'count-only' }),
      historyItem({ id: 'names-only' }),
      historyItem({ id: 'old-record' }),
    ]);
    api.details.set('count-only', historyDetail({ id: 'count-only', participantCount: 1, participantNames: [] }));
    api.details.set('names-only', historyDetail({ id: 'names-only', participantCount: null, participantNames: ['João', 'Maria'] }));
    api.details.set('old-record', historyDetail({ id: 'old-record', participantCount: null, participantNames: [] }));
    const { fixture, component } = await renderPage(api.items, api);

    await component.openDetail(api.items[0]);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Quantidade: 1 aluno');

    await component.openDetail(api.items[1]);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Participantes:');
    expect(fixture.nativeElement.textContent).toContain('João');
    expect(fixture.nativeElement.textContent).toContain('Maria');

    await component.openDetail(api.items[2]);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Participação não informada.');
  });

  it('handles invalid range and missing detail errors safely', async () => {
    const api = new FakeHistoryApi([historyItem({ id: 'history-1' })]);
    api.listError = new HttpErrorResponse({ status: 400, statusText: 'Bad Request' });
    const { fixture, component } = await renderPage([], api);

    await component.loadHistory();
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Período inválido.');

    api.listError = null;
    api.detailError = new HttpErrorResponse({ status: 404, statusText: 'Not Found' });
    await component.openDetail(historyItem({ id: 'missing' }));
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Registro de histórico não encontrado.');
  });
});

async function renderPage(items: HistoryListItem[], api = new FakeHistoryApi(items)) {
  TestBed.resetTestingModule();

  await TestBed.configureTestingModule({
    imports: [HistoryPage],
    providers: [
      provideRouter([]),
      { provide: HistoryApiService, useValue: api },
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(HistoryPage);
  const component = fixture.componentInstance as any;
  fixture.detectChanges();
  await fixture.whenStable();
  await component.loadHistory();
  fixture.detectChanges();

  return { fixture, component, api };
}

function historyItem(overrides: Partial<HistoryListItem> = {}): HistoryListItem {
  return {
    id: 'history-id',
    trainingDate: '2026-08-24',
    workoutName: 'Treino Snapshot',
    blockCount: 2,
    participantCount: null,
    completedByName: 'Professor Snapshot',
    completedAt: '2026-08-24T15:32:00Z',
    scheduleEntryId: 'schedule-id',
    ...overrides,
  };
}

function historyDetail(overrides: Partial<HistoryDetail> = {}): HistoryDetail {
  return {
    id: 'history-id',
    trainingDate: '2026-08-24',
    workoutName: 'Treino Snapshot',
    participantCount: null,
    participantNames: [],
    completedByName: 'Professor Snapshot',
    completedAt: '2026-08-24T15:32:00Z',
    blocks: [
      { position: 1, blockName: 'Bloco Snapshot', categoryName: 'Categoria Snapshot' },
    ],
    ...overrides,
  };
}

class FakeHistoryApi {
  ranges: Array<{ from: string; to: string }> = [];
  loadedDetails: string[] = [];
  details = new Map<string, HistoryDetail>();
  listError: unknown = null;
  detailError: unknown = null;

  constructor(public items: HistoryListItem[]) {}

  async list(from: string, to: string): Promise<HistoryListItem[]> {
    this.ranges.push({ from, to });
    if (this.listError) {
      throw this.listError;
    }
    return this.items;
  }

  async getById(id: string): Promise<HistoryDetail> {
    this.loadedDetails.push(id);
    if (this.detailError) {
      throw this.detailError;
    }
    return this.details.get(id) ?? historyDetail({ id });
  }
}
