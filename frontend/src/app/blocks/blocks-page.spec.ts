import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { Category, CategoryApiService } from '../categories/category-api.service';
import { Block, BlockApiService, BlockRequest } from './block-api.service';
import { BlocksPage } from './blocks-page';

describe('BlocksPage', () => {
  it('shows empty state', async () => {
    const { fixture } = await renderPage([], [category()]);

    expect(fixture.nativeElement.textContent).toContain('Nenhum bloco cadastrado.');
  });

  it('shows populated block list with category', async () => {
    const { fixture } = await renderPage([block({ name: 'Base', category: { id: 'cat-1', name: 'Mobilidade' } })], [category({ id: 'cat-1', name: 'Mobilidade' })]);
    const text = fixture.nativeElement.textContent;

    expect(text).toContain('Base');
    expect(text).toContain('Mobilidade');
    expect(text).toContain('Ativo');
    expect(text).not.toContain('block-id');
    expect(text).not.toContain('cat-1');
  });

  it('keeps operational block controls visible for authenticated users', async () => {
    const { fixture } = await renderPage([block({ name: 'Base' })], [category()]);

    expect(fixture.nativeElement.textContent).toContain('Novo bloco');
    expect(fixture.nativeElement.querySelector('button[aria-label="Editar bloco"]')).toBeTruthy();
    expect(fixture.nativeElement.querySelector('button[aria-label="Desativar bloco"]')).toBeTruthy();
    expect(fixture.nativeElement.querySelector('button[aria-label="Excluir bloco"]')).toBeTruthy();
  });

  it('creates block and handles duplicate name', async () => {
    const blockApi = new FakeBlockApi([]);
    const { fixture, component } = await renderPage([], [category()], blockApi);

    component.openCreateDialog();
    component.createForm = { name: 'Base', categoryId: 'category-id', description: 'Instruções gerais', sequence: ['Item A'], sequenceText: '' };
    await component.createBlock();
    fixture.detectChanges();

    expect(blockApi.created).toEqual([{ name: 'Base', categoryId: 'category-id', description: 'Instruções gerais', sequence: [{ text: 'Item A' }] }]);
    expect(component.createForm).toEqual({ name: '', categoryId: '', description: '', sequence: [], sequenceText: '' });
    expect(component.blocks()).toHaveLength(1);
    expect(fixture.nativeElement.textContent).toContain('Bloco cadastrado com sucesso.');

    blockApi.createError = new HttpErrorResponse({ status: 409, statusText: 'Conflict' });
    component.openCreateDialog();
    component.createForm = { name: 'base', categoryId: 'category-id', description: '', sequence: [], sequenceText: '' };
    await component.createBlock();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Já existe um bloco com este nome nesta categoria.');
  });

  it('creates a generic block without optional description or sequence', async () => {
    const blockApi = new FakeBlockApi([]);
    const { component } = await renderPage([], [category()], blockApi);

    component.openCreateDialog();
    component.createForm = { name: 'Aquecimento livre', categoryId: 'category-id', description: '', sequence: [], sequenceText: '' };
    await component.createBlock();

    expect(blockApi.created[0]).toEqual({ name: 'Aquecimento livre', categoryId: 'category-id', description: '', sequence: [] });
  });

  it('manages free-text sequence items with enter, ordering, removal and duplicates', async () => {
    const { component } = await renderPage([], [category()]);
    const input = document.createElement('input');
    let prevented = false;

    input.value = 'Jab';
    component.addSequenceItemFromKeyboard({ preventDefault: () => { prevented = true; } }, component.createForm, input);
    input.value = 'Direto';
    component.addSequenceItem(component.createForm, input);
    input.value = 'Jab';
    component.addSequenceItem(component.createForm, input);
    expect(component.createForm.sequence).toEqual(['Jab', 'Direto', 'Jab']);
    component.moveSequenceItemUp(component.createForm, 1);
    component.moveSequenceItemDown(component.createForm, 0);
    component.removeSequenceItem(component.createForm, 2);

    expect(prevented).toBe(true);
    expect(component.createForm.sequence).toEqual(['Jab', 'Direto']);
    expect(component.createForm.sequenceText).toBe('');
  });

  it('prevents creation when there are no active categories', async () => {
    const { fixture, component } = await renderPage([], [category({ active: false })]);

    component.openCreateDialog();
    fixture.detectChanges();

    expect(component.createDialogOpen).toBe(false);
    expect(fixture.nativeElement.textContent).toContain('Cadastre ou ative uma categoria antes de criar blocos.');
  });

  it('edits block and changes category', async () => {
    const original = block({ id: 'block-1', name: 'Base', category: { id: 'cat-1', name: 'Técnica' } });
    const blockApi = new FakeBlockApi([original]);
    const { fixture, component } = await renderPage(
      [original],
      [category({ id: 'cat-1', name: 'Técnica' }), category({ id: 'cat-2', name: 'Mobilidade' })],
      blockApi,
    );

    component.openEditDialog(original);
    component.editForm = { name: 'Avançado', categoryId: 'cat-2', description: '', sequence: [], sequenceText: '' };
    await component.updateBlock();
    fixture.detectChanges();

    expect(blockApi.updated[0]).toEqual({ id: 'block-1', request: { name: 'Avançado', categoryId: 'cat-2', description: '', sequence: [] } });
    expect(fixture.nativeElement.textContent).toContain('Avançado');
    expect(fixture.nativeElement.textContent).toContain('Mobilidade');
  });

  it('loads existing description and ordered sequence for editing', async () => {
    const original = block({
      id: 'block-1',
      description: 'Executar em dupla.',
      sequence: [
        { position: 1, text: 'Defende' },
        { position: 2, text: 'Responde' },
      ],
    });
    const { component } = await renderPage([original], [category()]);

    component.openEditDialog(original);

    expect(component.editForm.description).toBe('Executar em dupla.');
    expect(component.editForm.sequence).toEqual(['Defende', 'Responde']);
  });

  it('requires an active category when editing a block whose category became inactive', async () => {
    const original = block({ id: 'block-1', category: { id: 'cat-1', name: 'Inativa' } });
    const { fixture, component } = await renderPage([original], [category({ id: 'cat-1', name: 'Inativa', active: false })]);

    component.openEditDialog(original);

    expect(component.editForm.categoryId).toBe('');
    expect(component.formValid(component.editForm)).toBe(false);
    expect(component.categoryIsActive(original.category.id)).toBe(false);
  });

  it('deactivates and reactivates block', async () => {
    const original = block({ id: 'block-1', active: true });
    const blockApi = new FakeBlockApi([original]);
    const { fixture, component } = await renderPage([original], [category()], blockApi);

    component.confirmStatus(original);
    expect(component.confirmationText()).toContain('indisponível para uso futuro');
    await component.applyConfirmation();
    fixture.detectChanges();

    expect(blockApi.statusChanges[0]).toEqual({ id: 'block-1', active: false });
    expect(fixture.nativeElement.textContent).toContain('Inativo');

    const inactive = component.blocks()[0];
    component.confirmStatus(inactive);
    await component.applyConfirmation();

    expect(blockApi.statusChanges[1]).toEqual({ id: 'block-1', active: true });
  });

  it('deletes block after explicit confirmation', async () => {
    const original = block({ id: 'block-1', name: 'Base' });
    const blockApi = new FakeBlockApi([original]);
    const { fixture, component } = await renderPage([original], [category()], blockApi);

    component.confirmDelete(original);
    expect(component.confirmationTitle()).toBe('Excluir Base?');
    expect(component.confirmationText()).toContain('permanentemente removido');
    await component.applyConfirmation();
    fixture.detectChanges();

    expect(blockApi.deleted).toEqual(['block-1']);
    expect(fixture.nativeElement.textContent).toContain('Nenhum bloco cadastrado.');
  });

  it('shows safe message when block delete is blocked by workouts', async () => {
    const original = block({ id: 'block-1', name: 'Base' });
    const blockApi = new FakeBlockApi([original]);
    blockApi.deleteError = new HttpErrorResponse({ status: 409, statusText: 'Conflict' });
    const { fixture, component } = await renderPage([original], [category()], blockApi);

    component.confirmDelete(original);
    await component.applyConfirmation();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Este bloco está sendo utilizado por um ou mais treinos e não pode ser excluído.');
    expect(fixture.nativeElement.textContent).not.toContain('workout_blocks_block_id_fkey');
  });

  it('clears form state after dialog closes', async () => {
    const original = block({ id: 'block-1', name: 'Base' });
    const { component } = await renderPage([original], [category()]);

    component.openCreateDialog();
    component.createForm = { name: 'Temporário', categoryId: 'category-id', description: 'texto', sequence: ['um'], sequenceText: 'rascunho' };
    component.closeCreateDialog();
    expect(component.createForm).toEqual({ name: '', categoryId: '', description: '', sequence: [], sequenceText: '' });

    component.openEditDialog(original);
    component.editForm = { name: 'Temporário', categoryId: 'category-id', description: 'texto', sequence: ['um'], sequenceText: 'rascunho' };
    component.closeEditDialog();
    expect(component.editForm).toEqual({ name: '', categoryId: '', description: '', sequence: [], sequenceText: '' });
  });
});

async function renderPage(blocks: Block[], categories: Category[], blockApi = new FakeBlockApi(blocks)) {
  TestBed.resetTestingModule();

  await TestBed.configureTestingModule({
    imports: [BlocksPage],
    providers: [
      provideRouter([]),
      { provide: BlockApiService, useValue: blockApi },
      { provide: CategoryApiService, useValue: new FakeCategoryApi(categories) },
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(BlocksPage);
  const component = fixture.componentInstance as any;
  fixture.detectChanges();
  await fixture.whenStable();
  await component.loadData();
  fixture.detectChanges();

  return { fixture, component, blockApi };
}

function category(overrides: Partial<Category> = {}): Category {
  return {
    id: 'category-id',
    name: 'Categoria',
    active: true,
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

function block(overrides: Partial<Block> = {}): Block {
  return {
    id: 'block-id',
    name: 'Bloco',
    description: null,
    sequence: [],
    active: true,
    category: {
      id: 'category-id',
      name: 'Categoria',
    },
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

class FakeBlockApi {
  created: BlockRequest[] = [];
  updated: Array<{ id: string; request: BlockRequest }> = [];
  statusChanges: Array<{ id: string; active: boolean }> = [];
  deleted: string[] = [];
  createError: unknown = null;
  deleteError: unknown = null;

  constructor(private blocks: Block[]) {}

  async list(): Promise<Block[]> {
    return this.blocks;
  }

  async create(request: BlockRequest): Promise<Block> {
    this.created.push(request);
    if (this.createError) {
      throw this.createError;
    }

    const created = block({
      id: `block-${this.created.length}`,
      name: request.name,
      description: request.description ?? null,
      sequence: (request.sequence ?? []).map((item, index) => ({ position: index + 1, text: item.text })),
      active: true,
      category: { id: request.categoryId, name: request.categoryId === 'cat-2' ? 'Mobilidade' : 'Categoria' },
    });
    this.blocks = [...this.blocks, created];
    return created;
  }

  async update(id: string, request: BlockRequest): Promise<Block> {
    this.updated.push({ id, request });
    const current = this.blocks.find((item) => item.id === id) ?? block({ id });
    const updated = {
      ...current,
      name: request.name,
      description: request.description ?? null,
      sequence: (request.sequence ?? []).map((item, index) => ({ position: index + 1, text: item.text })),
      category: { id: request.categoryId, name: request.categoryId === 'cat-2' ? 'Mobilidade' : current.category.name },
    };
    this.blocks = this.blocks.map((item) => item.id === id ? updated : item);
    return updated;
  }

  async setStatus(id: string, active: boolean): Promise<Block> {
    this.statusChanges.push({ id, active });
    const current = this.blocks.find((item) => item.id === id) ?? block({ id });
    const updated = { ...current, active };
    this.blocks = this.blocks.map((item) => item.id === id ? updated : item);
    return updated;
  }

  async delete(id: string): Promise<void> {
    this.deleted.push(id);
    if (this.deleteError) {
      throw this.deleteError;
    }
    this.blocks = this.blocks.filter((item) => item.id !== id);
  }
}

class FakeCategoryApi {
  constructor(private categories: Category[]) {}

  async list(): Promise<Category[]> {
    return this.categories;
  }
}
