import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { ProfessorApiService } from './professor-api.service';

describe('ProfessorApiService', () => {
  let service: ProfessorApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });

    service = TestBed.inject(ProfessorApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    http.verify();
  });

  it('uses expected professor API endpoints with credentials', async () => {
    const list = service.list();
    let request = http.expectOne('/api/professors');
    expect(request.request.method).toBe('GET');
    expect(request.request.withCredentials).toBe(true);
    request.flush([]);
    await list;

    const create = service.create({ name: 'Ana', email: 'ana@example.com', password: 'uma senha longa e segura' });
    request = http.expectOne('/api/professors');
    expect(request.request.method).toBe('POST');
    expect(request.request.withCredentials).toBe(true);
    request.flush(professorResponse('professor-id'));
    await create;

    const update = service.update('professor-id', { name: 'Ana Maria', email: 'ana@example.com' });
    request = http.expectOne('/api/professors/professor-id');
    expect(request.request.method).toBe('PUT');
    expect(request.request.withCredentials).toBe(true);
    request.flush(professorResponse('professor-id'));
    await update;

    const status = service.setStatus('professor-id', false);
    request = http.expectOne('/api/professors/professor-id/status');
    expect(request.request.method).toBe('PATCH');
    expect(request.request.body).toEqual({ active: false });
    expect(request.request.withCredentials).toBe(true);
    request.flush(professorResponse('professor-id'));
    await status;

    const password = service.setPassword('professor-id', 'outra senha longa segura');
    request = http.expectOne('/api/professors/professor-id/password');
    expect(request.request.method).toBe('PUT');
    expect(request.request.body).toEqual({ password: 'outra senha longa segura' });
    expect(request.request.withCredentials).toBe(true);
    request.flush(null, { status: 204, statusText: 'No Content' });
    await password;

    const remove = service.delete('professor-id');
    request = http.expectOne('/api/professors/professor-id');
    expect(request.request.method).toBe('DELETE');
    expect(request.request.withCredentials).toBe(true);
    request.flush(null, { status: 204, statusText: 'No Content' });
    await remove;
  });
});

function professorResponse(id: string) {
  return {
    id,
    name: 'Ana',
    email: 'ana@example.com',
    active: true,
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
  };
}
