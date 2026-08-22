import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { AuthService } from './auth.service';
import { AuthUser } from './auth.models';

const adminUser: AuthUser = {
  id: 'admin-id',
  name: 'Admin',
  email: 'admin@example.com',
  role: 'ADMIN',
};

describe('AuthService', () => {
  let service: AuthService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });

    service = TestBed.inject(AuthService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    http.verify();
  });

  it('sets authenticated state after successful login', async () => {
    const promise = service.login('admin@example.com', 'senha longa segura');
    const request = http.expectOne('/api/auth/login');

    expect(request.request.method).toBe('POST');
    expect(request.request.withCredentials).toBe(true);
    request.flush(adminUser);

    await expect(promise).resolves.toEqual(adminUser);
    expect(service.currentUser()).toEqual(adminUser);
    expect(service.status()).toBe('authenticated');
  });

  it('sets unauthenticated state after failed login', async () => {
    const promise = service.login('admin@example.com', 'senha errada');
    const request = http.expectOne('/api/auth/login');

    request.flush({ error: 'invalid email or password' }, { status: 401, statusText: 'Unauthorized' });

    await expect(promise).rejects.toBeTruthy();
    expect(service.currentUser()).toBeNull();
    expect(service.status()).toBe('unauthenticated');
  });

  it('restores an existing session', async () => {
    const promise = service.restoreSession();
    const request = http.expectOne('/api/auth/me');

    expect(request.request.method).toBe('GET');
    expect(request.request.withCredentials).toBe(true);
    request.flush(adminUser);

    await expect(promise).resolves.toEqual(adminUser);
    expect(service.currentUser()).toEqual(adminUser);
    expect(service.status()).toBe('authenticated');
  });

  it('treats 401 restoration as unauthenticated state', async () => {
    const promise = service.restoreSession();
    const request = http.expectOne('/api/auth/me');

    request.flush({ error: 'authentication required' }, { status: 401, statusText: 'Unauthorized' });

    await expect(promise).resolves.toBeNull();
    expect(service.currentUser()).toBeNull();
    expect(service.status()).toBe('unauthenticated');
  });

  it('logs out and clears auth state', async () => {
    const login = service.login('admin@example.com', 'senha longa segura');
    http.expectOne('/api/auth/login').flush(adminUser);
    await login;

    const logout = service.logout();
    const request = http.expectOne('/api/auth/logout');

    expect(request.request.method).toBe('POST');
    expect(request.request.withCredentials).toBe(true);
    request.flush(null, { status: 204, statusText: 'No Content' });

    await expect(logout).resolves.toBeUndefined();
    expect(service.currentUser()).toBeNull();
    expect(service.status()).toBe('unauthenticated');
  });
});
