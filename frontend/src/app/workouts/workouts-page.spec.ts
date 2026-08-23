import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { Block, BlockApiService } from '../blocks/block-api.service';
import { WorkoutApiService, WorkoutDetail, WorkoutListItem } from './workout-api.service';
import { WorkoutsPage } from './workouts-page';

describe('WorkoutsPage', () => {
  it('shows empty state', async () => {
    const { fixture } = await renderPage([], [block()]);

    expect(fixture.nativeElement.textContent).toContain('Nenhum treino cadastrado.');
  });

  it('shows populated workout list with block count', async () => {
    const { fixture } = await renderPage([workout({ name: 'Treino Base', blockCount: 4 })], [block()]);
    const text = fixture.nativeElement.textContent;

    expect(text).toContain('Treino Base');
    expect(text).toContain('4 blocos');
    expect(text).toContain('Ativo');
    expect(text).not.toContain('workout-id');
  });

  it('keeps operational workout controls visible for authenticated users', async () => {
    const { fixture } = await renderPage([workout({ name: 'Treino Base' })], [block()]);

    expect(fixture.nativeElement.textContent).toContain('Novo treino');
    expect(fixture.nativeElement.querySelector('button[aria-label="Editar treino"]')).toBeTruthy();
    expect(fixture.nativeElement.querySelector('button[aria-label="Desativar treino"]')).toBeTruthy();
    expect(fixture.nativeElement.querySelector('button[aria-label="Excluir treino"]')).toBeTruthy();
  });

  it('creates workout and preserves selected order', async () => {
    const api = new FakeWorkoutApi([]);
    const { fixture, component } = await renderPage([], [block({ id: 'block-1', name: 'Primeiro' }), block({ id: 'block-2', name: 'Segundo' })], api);

    component.openCreateDialog();
    component.createForm.name = 'Treino';
    component.createForm.selectedBlockId = 'block-1';
    component.addSelectedBlock(component.createForm);
    component.createForm.selectedBlockId = 'block-2';
    component.addSelectedBlock(component.createForm);
    component.moveBlockUp(component.createForm, 1);
    await component.createWorkout();
    fixture.detectChanges();

    expect(api.created[0]).toEqual({ name: 'Treino', blockIds: ['block-2', 'block-1'] });
    expect(component.createForm).toEqual({ name: '', selectedBlockId: '', blocks: [] });
    expect(fixture.nativeElement.textContent).toContain('Treino cadastrado com sucesso.');
  });

  it('prevents creation when there are no active blocks', async () => {
    const { fixture, component } = await renderPage([], [block({ active: false })]);

    component.openCreateDialog();
    fixture.detectChanges();

    expect(component.createDialogOpen).toBe(false);
    expect(fixture.nativeElement.textContent).toContain('Cadastre ou ative pelo menos um bloco antes de criar um treino.');
  });

  it('prevents duplicate selection and removes blocks', async () => {
    const { component } = await renderPage([], [block({ id: 'block-1' })]);

    component.openCreateDialog();
    component.createForm.selectedBlockId = 'block-1';
    component.addSelectedBlock(component.createForm);
    component.createForm.selectedBlockId = 'block-1';
    component.addSelectedBlock(component.createForm);

    expect(component.createForm.blocks).toHaveLength(1);
    expect(component.availableBlocks(component.createForm)).toHaveLength(0);

    component.removeBlock(component.createForm, 0);
    expect(component.createForm.blocks).toHaveLength(0);
  });

  it('loads detail for edit and preserves order changes', async () => {
    const api = new FakeWorkoutApi([workout({ id: 'workout-1', name: 'Treino' })]);
    api.details.set('workout-1', workoutDetail({
      id: 'workout-1',
      name: 'Treino',
      blocks: [
        workoutBlock({ id: 'block-1', name: 'Primeiro', position: 1 }),
        workoutBlock({ id: 'block-2', name: 'Segundo', position: 2 }),
      ],
    }));
    const { component } = await renderPage(api.workouts, [block({ id: 'block-1' }), block({ id: 'block-2' })], api);

    await component.openEditDialog(api.workouts[0]);
    component.editForm.name = 'Editado';
    component.moveBlockUp(component.editForm, 1);
    await component.updateWorkout();

    expect(api.loadedDetails).toEqual(['workout-1']);
    expect(api.updated[0]).toEqual({ id: 'workout-1', request: { name: 'Editado', blockIds: ['block-2', 'block-1'] } });
  });

  it('displays and preserves existing inactive block while keeping it unavailable for new selection', async () => {
    const api = new FakeWorkoutApi([workout({ id: 'workout-1' })]);
    api.details.set('workout-1', workoutDetail({
      id: 'workout-1',
      blocks: [workoutBlock({ id: 'inactive-block', active: false, position: 1 })],
    }));
    const inactive = block({ id: 'inactive-block', active: false });
    const active = block({ id: 'active-block', active: true });
    const { component } = await renderPage(api.workouts, [inactive, active], api);

    await component.openEditDialog(api.workouts[0]);

    expect(component.editForm.blocks[0].id).toBe('inactive-block');
    expect(component.editForm.blocks[0].active).toBe(false);
    expect(component.availableBlocks(component.editForm).map((item: Block) => item.id)).toEqual(['active-block']);

    await component.updateWorkout();
    expect(api.updated[0].request.blockIds).toEqual(['inactive-block']);
  });

  it('handles duplicate workout name', async () => {
    const api = new FakeWorkoutApi([]);
    api.createError = new HttpErrorResponse({ status: 409, statusText: 'Conflict' });
    const { fixture, component } = await renderPage([], [block()], api);

    component.openCreateDialog();
    component.createForm.name = 'Treino';
    component.createForm.selectedBlockId = 'block-id';
    component.addSelectedBlock(component.createForm);
    await component.createWorkout();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Já existe um treino com este nome.');
  });

  it('deactivates and reactivates workout', async () => {
    const original = workout({ id: 'workout-1', active: true });
    const api = new FakeWorkoutApi([original]);
    const { fixture, component } = await renderPage([original], [block()], api);

    component.confirmStatus(original);
    expect(component.confirmationText()).toContain('indisponível para uso futuro');
    await component.applyConfirmation();
    fixture.detectChanges();

    expect(api.statusChanges[0]).toEqual({ id: 'workout-1', active: false });
    expect(fixture.nativeElement.textContent).toContain('Inativo');

    const inactive = component.workouts()[0];
    component.confirmStatus(inactive);
    await component.applyConfirmation();

    expect(api.statusChanges[1]).toEqual({ id: 'workout-1', active: true });
  });

  it('deletes workout after explicit confirmation', async () => {
    const original = workout({ id: 'workout-1', name: 'Treino Base' });
    const api = new FakeWorkoutApi([original]);
    const { fixture, component } = await renderPage([original], [block()], api);

    component.confirmDelete(original);
    expect(component.confirmationTitle()).toBe('Excluir Treino Base?');
    expect(component.confirmationText()).toContain('permanentemente removido');
    await component.applyConfirmation();
    fixture.detectChanges();

    expect(api.deleted).toEqual(['workout-1']);
    expect(fixture.nativeElement.textContent).toContain('Nenhum treino cadastrado.');
  });

  it('shows safe message when workout delete is blocked by agenda', async () => {
    const original = workout({ id: 'workout-1', name: 'Treino Base' });
    const api = new FakeWorkoutApi([original]);
    api.deleteError = new HttpErrorResponse({ status: 409, statusText: 'Conflict' });
    const { fixture, component } = await renderPage([original], [block()], api);

    component.confirmDelete(original);
    await component.applyConfirmation();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Este treino está sendo utilizado na agenda e não pode ser excluído.');
    expect(fixture.nativeElement.textContent).not.toContain('schedule_entries_workout_id_fkey');
  });

  it('clears form state after dialog closes', async () => {
    const original = workout({ id: 'workout-1', name: 'Treino' });
    const { component } = await renderPage([original], [block()]);

    component.openCreateDialog();
    component.createForm.name = 'Temporário';
    component.createForm.blocks = [workoutBlock()];
    component.closeCreateDialog();
    expect(component.createForm).toEqual({ name: '', selectedBlockId: '', blocks: [] });

    await component.openEditDialog(original);
    component.editForm.name = 'Temporário';
    component.closeEditDialog();
    expect(component.editForm).toEqual({ name: '', selectedBlockId: '', blocks: [] });
  });
});

async function renderPage(workouts: WorkoutListItem[], blocks: Block[], api = new FakeWorkoutApi(workouts)) {
  TestBed.resetTestingModule();

  await TestBed.configureTestingModule({
    imports: [WorkoutsPage],
    providers: [
      provideRouter([]),
      { provide: WorkoutApiService, useValue: api },
      { provide: BlockApiService, useValue: new FakeBlockApi(blocks) },
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(WorkoutsPage);
  const component = fixture.componentInstance as any;
  fixture.detectChanges();
  await fixture.whenStable();
  await component.loadData();
  fixture.detectChanges();

  return { fixture, component, api };
}

function workout(overrides: Partial<WorkoutListItem> = {}): WorkoutListItem {
  return {
    id: 'workout-id',
    name: 'Treino',
    active: true,
    blockCount: 1,
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

function workoutDetail(overrides: Partial<WorkoutDetail> = {}): WorkoutDetail {
  return {
    id: 'workout-id',
    name: 'Treino',
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
    name: 'Bloco',
    active: true,
    position: 1,
    category: { id: 'category-id', name: 'Categoria' },
    ...overrides,
  };
}

function block(overrides: Partial<Block> = {}): Block {
  return {
    id: 'block-id',
    name: 'Bloco',
    active: true,
    category: { id: 'category-id', name: 'Categoria' },
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

class FakeWorkoutApi {
  created: Array<{ name: string; blockIds: string[] }> = [];
  updated: Array<{ id: string; request: { name: string; blockIds: string[] } }> = [];
  statusChanges: Array<{ id: string; active: boolean }> = [];
  deleted: string[] = [];
  loadedDetails: string[] = [];
  details = new Map<string, WorkoutDetail>();
  createError: unknown = null;
  deleteError: unknown = null;

  constructor(public workouts: WorkoutListItem[]) {}

  async list(): Promise<WorkoutListItem[]> {
    return this.workouts;
  }

  async getById(id: string): Promise<WorkoutDetail> {
    this.loadedDetails.push(id);
    return this.details.get(id) ?? workoutDetail({ id });
  }

  async create(request: { name: string; blockIds: string[] }): Promise<WorkoutDetail> {
    this.created.push(request);
    if (this.createError) {
      throw this.createError;
    }
    const created = workoutDetail({
      id: `workout-${this.created.length}`,
      name: request.name,
      blocks: request.blockIds.map((id, index) => workoutBlock({ id, position: index + 1 })),
    });
    this.workouts = [...this.workouts, workout({ id: created.id, name: created.name, blockCount: created.blocks.length })];
    return created;
  }

  async update(id: string, request: { name: string; blockIds: string[] }): Promise<WorkoutDetail> {
    this.updated.push({ id, request });
    const updated = workoutDetail({
      id,
      name: request.name,
      blocks: request.blockIds.map((blockId, index) => workoutBlock({ id: blockId, position: index + 1 })),
    });
    this.workouts = this.workouts.map((item) => item.id === id ? workout({ ...item, name: request.name, blockCount: request.blockIds.length }) : item);
    return updated;
  }

  async setStatus(id: string, active: boolean): Promise<WorkoutListItem> {
    this.statusChanges.push({ id, active });
    const current = this.workouts.find((item) => item.id === id) ?? workout({ id });
    const updated = { ...current, active };
    this.workouts = this.workouts.map((item) => item.id === id ? updated : item);
    return updated;
  }

  async delete(id: string): Promise<void> {
    this.deleted.push(id);
    if (this.deleteError) {
      throw this.deleteError;
    }
    this.workouts = this.workouts.filter((item) => item.id !== id);
  }
}

class FakeBlockApi {
  constructor(private blocks: Block[]) {}

  async list(): Promise<Block[]> {
    return this.blocks;
  }
}
