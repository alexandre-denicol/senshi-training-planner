import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { Category, CategoryApiService } from './category-api.service';
import { CategoriesPage } from './categories-page';

describe('CategoriesPage', () => {
  it('shows empty state', async () => {
    const { fixture } = await renderPage([]);

    expect(fixture.nativeElement.textContent).toContain('Você ainda não cadastrou nenhuma categoria.');
  });

  it('shows populated category list', async () => {
    const { fixture } = await renderPage([category({ name: 'Técnica' })]);
    const text = fixture.nativeElement.textContent;

    expect(text).toContain('Técnica');
    expect(text).toContain('Ativa');
    expect(text).not.toContain('category-id');
  });

  it('keeps operational category controls visible for authenticated users', async () => {
    const { fixture } = await renderPage([category({ name: 'Técnica' })]);

    expect(fixture.nativeElement.textContent).toContain('Nova categoria');
    expect(fixture.nativeElement.querySelector('button[aria-label="Editar categoria"]')).toBeTruthy();
    expect(fixture.nativeElement.querySelector('button[aria-label="Desativar categoria"]')).toBeTruthy();
    expect(fixture.nativeElement.querySelector('button[aria-label="Excluir categoria"]')).toBeTruthy();
  });

  it('creates category and handles duplicate name', async () => {
    const api = new FakeCategoryApi([]);
    const { fixture, component } = await renderPage([], api);

    component.openCreateDialog();
    component.createForm = { name: 'Técnica' };
    await component.createCategory();
    fixture.detectChanges();

    expect(api.created).toEqual([{ name: 'Técnica' }]);
    expect(component.createForm.name).toBe('');
    expect(component.categories()).toHaveLength(1);
    expect(fixture.nativeElement.textContent).toContain('Categoria cadastrada com sucesso.');

    api.createError = new HttpErrorResponse({ status: 409, statusText: 'Conflict' });
    component.openCreateDialog();
    component.createForm = { name: 'técnica' };
    await component.createCategory();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Já existe uma categoria com este nome.');
  });

  it('edits category', async () => {
    const original = category({ id: 'cat-1', name: 'Técnica' });
    const api = new FakeCategoryApi([original]);
    const { fixture, component } = await renderPage([original], api);

    component.openEditDialog(original);
    component.editForm = { name: 'Mobilidade' };
    await component.updateCategory();
    fixture.detectChanges();

    expect(api.updated[0]).toEqual({ id: 'cat-1', request: { name: 'Mobilidade' } });
    expect(fixture.nativeElement.textContent).toContain('Mobilidade');
  });

  it('deactivates and reactivates category', async () => {
    const original = category({ id: 'cat-1', active: true });
    const api = new FakeCategoryApi([original]);
    const { fixture, component } = await renderPage([original], api);

    component.confirmStatus(original);
    expect(component.confirmationText()).toContain('indisponível para uso futuro');
    await component.applyConfirmation();
    fixture.detectChanges();

    expect(api.statusChanges[0]).toEqual({ id: 'cat-1', active: false });
    expect(fixture.nativeElement.textContent).toContain('Inativa');

    const inactive = component.categories()[0];
    component.confirmStatus(inactive);
    await component.applyConfirmation();

    expect(api.statusChanges[1]).toEqual({ id: 'cat-1', active: true });
  });

  it('deletes category after explicit confirmation', async () => {
    const original = category({ id: 'cat-1', name: 'Técnica' });
    const api = new FakeCategoryApi([original]);
    const { fixture, component } = await renderPage([original], api);

    component.confirmDelete(original);
    expect(component.confirmationTitle()).toBe('Excluir Técnica?');
    expect(component.confirmationText()).toContain('permanentemente removida');
    await component.applyConfirmation();
    fixture.detectChanges();

    expect(api.deleted).toEqual(['cat-1']);
    expect(fixture.nativeElement.textContent).toContain('Você ainda não cadastrou nenhuma categoria.');
  });

  it('shows safe message when category delete is blocked by blocks', async () => {
    const original = category({ id: 'cat-1', name: 'Técnica' });
    const api = new FakeCategoryApi([original]);
    api.deleteError = new HttpErrorResponse({ status: 409, statusText: 'Conflict' });
    const { fixture, component } = await renderPage([original], api);

    component.confirmDelete(original);
    await component.applyConfirmation();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Esta categoria está sendo utilizada por um ou mais blocos e não pode ser excluída.');
    expect(fixture.nativeElement.textContent).not.toContain('blocks_category_id_fkey');
  });

  it('clears form state after dialog closes', async () => {
    const original = category({ id: 'cat-1', name: 'Técnica' });
    const { component } = await renderPage([original]);

    component.openCreateDialog();
    component.createForm.name = 'Temporária';
    component.closeCreateDialog();
    expect(component.createForm.name).toBe('');

    component.openEditDialog(original);
    component.editForm.name = 'Temporária';
    component.closeEditDialog();
    expect(component.editForm.name).toBe('');
  });
});

async function renderPage(initial: Category[], api = new FakeCategoryApi(initial)) {
  TestBed.resetTestingModule();

  await TestBed.configureTestingModule({
    imports: [CategoriesPage],
    providers: [
      provideRouter([]),
      { provide: CategoryApiService, useValue: api },
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(CategoriesPage);
  const component = fixture.componentInstance as any;
  fixture.detectChanges();
  await fixture.whenStable();
  fixture.detectChanges();

  return { fixture, component, api };
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

class FakeCategoryApi {
  created: Array<{ name: string }> = [];
  updated: Array<{ id: string; request: { name: string } }> = [];
  statusChanges: Array<{ id: string; active: boolean }> = [];
  deleted: string[] = [];
  createError: unknown = null;
  deleteError: unknown = null;

  constructor(private categories: Category[]) {}

  async list(): Promise<Category[]> {
    return this.categories;
  }

  async create(request: { name: string }): Promise<Category> {
    this.created.push(request);
    if (this.createError) {
      throw this.createError;
    }

    const created = category({
      id: `cat-${this.created.length}`,
      name: request.name,
      active: true,
    });
    this.categories = [...this.categories, created];
    return created;
  }

  async update(id: string, request: { name: string }): Promise<Category> {
    this.updated.push({ id, request });
    const current = this.categories.find((item) => item.id === id) ?? category({ id });
    const updated = { ...current, name: request.name };
    this.categories = this.categories.map((item) => item.id === id ? updated : item);
    return updated;
  }

  async setStatus(id: string, active: boolean): Promise<Category> {
    this.statusChanges.push({ id, active });
    const current = this.categories.find((item) => item.id === id) ?? category({ id });
    const updated = { ...current, active };
    this.categories = this.categories.map((item) => item.id === id ? updated : item);
    return updated;
  }

  async delete(id: string): Promise<void> {
    this.deleted.push(id);
    if (this.deleteError) {
      throw this.deleteError;
    }
    this.categories = this.categories.filter((item) => item.id !== id);
  }
}
