import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { AuthService } from '../auth/auth.service';
import { AuthUser } from '../auth/auth.models';
import { AppShell } from './app-shell';

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

    expect(fixture.nativeElement.textContent).toContain('Professores');
    expect(fixture.nativeElement.textContent).toContain('Categorias');
    expect(fixture.nativeElement.textContent).toContain('Blocos');
    expect(fixture.nativeElement.textContent).toContain('Treinos');
  });

  it('does not show management navigation for PROFESSOR', async () => {
    const fixture = await renderShell(professorUser);

    expect(fixture.nativeElement.textContent).not.toContain('Professores');
    expect(fixture.nativeElement.textContent).not.toContain('Categorias');
    expect(fixture.nativeElement.textContent).not.toContain('Blocos');
    expect(fixture.nativeElement.textContent).not.toContain('Treinos');
  });
});

async function renderShell(user: AuthUser) {
  TestBed.resetTestingModule();

  const currentUser = signal<AuthUser | null>(user);
  await TestBed.configureTestingModule({
    imports: [AppShell],
    providers: [
      {
        provide: AuthService,
        useValue: {
          currentUser: currentUser.asReadonly(),
          logout: () => Promise.resolve(),
        },
      },
      provideRouter([]),
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(AppShell);
  fixture.detectChanges();
  await fixture.whenStable();
  return fixture;
}
