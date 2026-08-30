import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { Professor, ProfessorApiService } from './professor-api.service';
import { ProfessorsPage } from './professors-page';

describe('ProfessorsPage', () => {
  it('shows empty list state', async () => {
    const { fixture } = await renderPage([]);

    expect(fixture.nativeElement.textContent).toContain('Você ainda não cadastrou nenhum professor.');
  });

  it('shows populated professor list without password fields', async () => {
    const { fixture } = await renderPage([professor({ name: 'Ana Silva', email: 'ana@example.com' })]);
    const text = fixture.nativeElement.textContent;

    expect(text).toContain('Ana Silva');
    expect(text).toContain('ana@example.com');
    expect(text).toContain('Ativo');
    expect(text).not.toContain('password');
    expect(text).not.toContain('hash');
  });

  it('creates professor and handles duplicate email', async () => {
    const api = new FakeProfessorApi([]);
    const { fixture, component } = await renderPage([], api);

    component.openCreateDialog();
    component.createForm = {
      name: 'Bruno',
      email: 'bruno@example.com',
      password: 'uma senha longa e segura',
      confirmPassword: 'uma senha longa e segura',
    };
    await component.createProfessor();
    fixture.detectChanges();

    expect(api.created[0].email).toBe('bruno@example.com');
    expect(fixture.nativeElement.textContent).toContain('Professor cadastrado com sucesso.');
    expect(component.createForm.password).toBe('');
    expect(component.professors()).toHaveLength(1);

    api.createError = new HttpErrorResponse({ status: 409, statusText: 'Conflict' });
    component.openCreateDialog();
    component.createForm = {
      name: 'Bruno',
      email: 'BRUNO@example.com',
      password: 'uma senha longa e segura',
      confirmPassword: 'uma senha longa e segura',
    };
    await component.createProfessor();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Já existe uma conta com este e-mail.');
  });

  it('edits professor', async () => {
    const original = professor({ id: 'prof-1', name: 'Ana Silva', email: 'ana@example.com' });
    const api = new FakeProfessorApi([original]);
    const { fixture, component } = await renderPage([original], api);

    component.openEditDialog(original);
    component.editForm = { name: 'Ana Maria', email: 'ana.maria@example.com' };
    await component.updateProfessor();
    fixture.detectChanges();

    expect(api.updated[0]).toEqual({
      id: 'prof-1',
      request: { name: 'Ana Maria', email: 'ana.maria@example.com' },
    });
    expect(fixture.nativeElement.textContent).toContain('Ana Maria');
  });

  it('activates and deactivates professor', async () => {
    const original = professor({ id: 'prof-1', active: true });
    const api = new FakeProfessorApi([original]);
    const { fixture, component } = await renderPage([original], api);

    component.confirmStatus(original);
    expect(component.confirmationText()).toContain('sessões ativas');
    await component.applyConfirmation();
    fixture.detectChanges();

    expect(api.statusChanges[0]).toEqual({ id: 'prof-1', active: false });
    expect(fixture.nativeElement.textContent).toContain('Inativo');

    const inactive = component.professors()[0];
    component.confirmStatus(inactive);
    await component.applyConfirmation();

    expect(api.statusChanges[1]).toEqual({ id: 'prof-1', active: true });
  });

  it('resets password and clears password state', async () => {
    const original = professor({ id: 'prof-1', name: 'Ana Silva' });
    const api = new FakeProfessorApi([original]);
    const { component } = await renderPage([original], api);

    component.openPasswordDialog(original);
    component.passwordForm = {
      password: 'uma senha longa e segura',
      confirmPassword: 'uma senha longa e segura',
    };
    await component.changePassword();

    expect(api.passwords[0]).toEqual({ id: 'prof-1', password: 'uma senha longa e segura' });
    expect(component.passwordForm.password).toBe('');
    expect(component.passwordForm.confirmPassword).toBe('');
  });

  it('deletes professor after explicit confirmation', async () => {
    const original = professor({ id: 'prof-1', name: 'Ana Silva' });
    const api = new FakeProfessorApi([original]);
    const { fixture, component } = await renderPage([original], api);

    component.confirmDelete(original);
    expect(component.confirmationTitle()).toBe('Excluir Ana Silva?');
    expect(component.confirmationText()).toContain('permanentemente removida');
    await component.applyConfirmation();
    fixture.detectChanges();

    expect(api.deleted).toEqual(['prof-1']);
    expect(fixture.nativeElement.textContent).toContain('Você ainda não cadastrou nenhum professor.');
  });

  it('clears passwords when dialogs close', async () => {
    const original = professor({ id: 'prof-1' });
    const { component } = await renderPage([original]);

    component.openCreateDialog();
    component.createForm.password = 'senha temporaria longa';
    component.createForm.confirmPassword = 'senha temporaria longa';
    component.closeCreateDialog();

    expect(component.createForm.password).toBe('');
    expect(component.createForm.confirmPassword).toBe('');

    component.openPasswordDialog(original);
    component.passwordForm.password = 'senha temporaria longa';
    component.passwordForm.confirmPassword = 'senha temporaria longa';
    component.closePasswordDialog();

    expect(component.passwordForm.password).toBe('');
    expect(component.passwordForm.confirmPassword).toBe('');
  });
});

async function renderPage(initial: Professor[], api = new FakeProfessorApi(initial)) {
  TestBed.resetTestingModule();

  await TestBed.configureTestingModule({
    imports: [ProfessorsPage],
    providers: [
      provideRouter([]),
      { provide: ProfessorApiService, useValue: api },
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(ProfessorsPage);
  const component = fixture.componentInstance as any;
  fixture.detectChanges();
  await fixture.whenStable();
  fixture.detectChanges();

  return { fixture, component, api };
}

function professor(overrides: Partial<Professor> = {}): Professor {
  return {
    id: 'professor-id',
    name: 'Professor',
    email: 'professor@example.com',
    active: true,
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
    ...overrides,
  };
}

class FakeProfessorApi {
  created: Array<{ name: string; email: string; password: string }> = [];
  updated: Array<{ id: string; request: { name: string; email: string } }> = [];
  statusChanges: Array<{ id: string; active: boolean }> = [];
  passwords: Array<{ id: string; password: string }> = [];
  deleted: string[] = [];
  createError: unknown = null;

  constructor(private professors: Professor[]) {}

  async list(): Promise<Professor[]> {
    return this.professors;
  }

  async create(request: { name: string; email: string; password: string }): Promise<Professor> {
    this.created.push(request);
    if (this.createError) {
      throw this.createError;
    }

    const created = professor({
      id: `prof-${this.created.length}`,
      name: request.name,
      email: request.email,
      active: true,
    });
    this.professors = [...this.professors, created];
    return created;
  }

  async update(id: string, request: { name: string; email: string }): Promise<Professor> {
    this.updated.push({ id, request });
    const updated = professor({ id, name: request.name, email: request.email });
    this.professors = this.professors.map((item) => item.id === id ? updated : item);
    return updated;
  }

  async setStatus(id: string, active: boolean): Promise<Professor> {
    this.statusChanges.push({ id, active });
    const current = this.professors.find((item) => item.id === id) ?? professor({ id });
    const updated = { ...current, active };
    this.professors = this.professors.map((item) => item.id === id ? updated : item);
    return updated;
  }

  async setPassword(id: string, password: string): Promise<void> {
    this.passwords.push({ id, password });
  }

  async delete(id: string): Promise<void> {
    this.deleted.push(id);
    this.professors = this.professors.filter((item) => item.id !== id);
  }
}
