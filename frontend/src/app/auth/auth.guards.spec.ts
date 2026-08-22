import { TestBed } from '@angular/core/testing';
import { provideRouter, Router, UrlTree } from '@angular/router';
import { AuthUser } from './auth.models';
import { AuthService } from './auth.service';
import { authGuard, loginGuard } from './auth.guards';

const adminUser: AuthUser = {
  id: 'admin-id',
  name: 'Admin',
  email: 'admin@example.com',
  role: 'ADMIN',
};

describe('auth guards', () => {
  it('allows authenticated access to /app', async () => {
    configureGuardTest(adminUser);

    const result = await TestBed.runInInjectionContext(() => authGuard({} as never, {} as never));

    expect(result).toBe(true);
  });

  it('redirects unauthenticated access to /login', async () => {
    configureGuardTest(null);

    const result = await TestBed.runInInjectionContext(() => authGuard({} as never, {} as never));

    expect((result as UrlTree).toString()).toBe('/login');
  });

  it('redirects authenticated access away from /login', async () => {
    configureGuardTest(adminUser);

    const result = await TestBed.runInInjectionContext(() => loginGuard({} as never, {} as never));

    expect((result as UrlTree).toString()).toBe('/app');
  });

  it('allows unauthenticated access to /login', async () => {
    configureGuardTest(null);

    const result = await TestBed.runInInjectionContext(() => loginGuard({} as never, {} as never));

    expect(result).toBe(true);
  });
});

function configureGuardTest(user: AuthUser | null): void {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      {
        provide: AuthService,
        useValue: {
          restoreSession: () => Promise.resolve(user),
        },
      },
    ],
  });

  TestBed.inject(Router);
}
