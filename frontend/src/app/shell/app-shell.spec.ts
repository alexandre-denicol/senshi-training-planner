import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { AuthService } from '../auth/auth.service';
import { AuthUser } from '../auth/auth.models';
import { AppShell } from './app-shell';
import { vi } from 'vitest';

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

describe('AppShell', () => {
  it('shows Professores navigation for ADMIN', async () => {
    const fixture = await renderShell(adminUser);

    expect(fixture.nativeElement.textContent).toContain('Dashboard');
    expect(fixture.nativeElement.textContent).toContain('Agenda');
    expect(fixture.nativeElement.textContent).toContain('Histórico');
    expect(fixture.nativeElement.textContent).toContain('Professores');
    expect(fixture.nativeElement.textContent).toContain('Categorias');
    expect(fixture.nativeElement.textContent).toContain('Blocos');
    expect(fixture.nativeElement.textContent).toContain('Treinos');
  });

  it('shows operational navigation but not Professores for PROFESSOR', async () => {
    const fixture = await renderShell(professorUser);

    expect(fixture.nativeElement.textContent).toContain('Dashboard');
    expect(fixture.nativeElement.textContent).toContain('Agenda');
    expect(fixture.nativeElement.textContent).toContain('Histórico');
    expect(fixture.nativeElement.textContent).toContain('Categorias');
    expect(fixture.nativeElement.textContent).toContain('Blocos');
    expect(fixture.nativeElement.textContent).toContain('Treinos');
    expect(fixture.nativeElement.textContent).not.toContain('Professores');
  });

  it('shows Alterar senha for ADMIN and PROFESSOR', async () => {
    for (const user of [adminUser, professorUser]) {
      const fixture = await renderShell(user);
      expect(fixture.nativeElement.querySelector('[aria-label="Alterar senha"]')).not.toBeNull();
    }
  });

  it('requires all password fields, eight characters, and matching confirmation', async () => {
    const fixture = await renderShell(adminUser);
    const component = fixture.componentInstance as any;

    component.openPasswordDialog();
    expect(component.passwordFormValid()).toBe(false);
    component.passwordForm.currentPassword = 'atual';
    expect(component.passwordFormValid()).toBe(false);
    component.passwordForm.newPassword = '1234567';
    component.passwordForm.confirmPassword = '1234567';
    expect(component.passwordFormValid()).toBe(false);
    component.passwordForm.newPassword = '12345678';
    expect(component.passwordFormValid()).toBe(false);
    component.passwordForm.confirmPassword = '87654321';
    expect(component.passwordFormValid()).toBe(false);
    component.passwordForm.confirmPassword = '12345678';
    expect(component.passwordFormValid()).toBe(true);
  });

  it('does not submit an invalid form', async () => {
    const auth = { currentUser: signal<AuthUser | null>(adminUser), logout: () => Promise.resolve(), changePassword: vi.fn() };
    const fixture = await renderShellWithAuth(auth);
    const component = fixture.componentInstance as any;
    component.passwordForm.newPassword = 'short';
    component.passwordForm.confirmPassword = 'short';

    await component.changePassword();
    expect(auth.changePassword).not.toHaveBeenCalled();
  });

  it('clears sensitive fields and logs out after a successful change', async () => {
    const auth = { currentUser: signal<AuthUser | null>(adminUser), logout: () => Promise.resolve(), changePassword: vi.fn().mockResolvedValue(undefined) };
    const fixture = await renderShellWithAuth(auth);
    const component = fixture.componentInstance as any;
    component.openPasswordDialog();
    component.passwordForm = { currentPassword: 'atual', newPassword: '12345678', confirmPassword: '12345678' };

    await component.changePassword();
    expect(auth.changePassword).toHaveBeenCalledWith('atual', '12345678');
    expect(component.passwordForm).toEqual({ currentPassword: '', newPassword: '', confirmPassword: '' });
  });

  it('keeps the dialog and clears no server state when the backend rejects the change', async () => {
    const auth = { currentUser: signal<AuthUser | null>(adminUser), logout: () => Promise.resolve(), changePassword: vi.fn().mockRejectedValue(new Error('backend details')) };
    const fixture = await renderShellWithAuth(auth);
    const component = fixture.componentInstance as any;
    component.openPasswordDialog();
    component.passwordForm = { currentPassword: 'atual', newPassword: '12345678', confirmPassword: '12345678' };

    await component.changePassword();
    expect(component.passwordDialogOpen()).toBe(true);
    expect(component.passwordError()).not.toContain('backend details');
    expect(component.passwordForm.currentPassword).toBe('atual');
  });
});

async function renderShell(user: AuthUser) {
  TestBed.resetTestingModule();

  const currentUser = signal<AuthUser | null>(user);
  return renderShellWithAuth({
    currentUser: currentUser.asReadonly(),
    logout: () => Promise.resolve(),
    changePassword: () => Promise.resolve(),
  });
}

async function renderShellWithAuth(auth: any) {
  TestBed.resetTestingModule();
  await TestBed.configureTestingModule({
    imports: [AppShell],
    providers: [
      {
        provide: AuthService,
        useValue: auth,
      },
      provideRouter([]),
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(AppShell);
  fixture.detectChanges();
  await fixture.whenStable();
  return fixture;
}
