import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { CategoryApiService } from './category-api.service';

describe('CategoryApiService', () => {
  let service: CategoryApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });

    service = TestBed.inject(CategoryApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    http.verify();
  });

  it('uses expected category API endpoints with credentials', async () => {
    const list = service.list();
    let request = http.expectOne('/api/categories');
    expect(request.request.method).toBe('GET');
    expect(request.request.withCredentials).toBe(true);
    request.flush([]);
    await list;

    const create = service.create({ name: 'Técnica' });
    request = http.expectOne('/api/categories');
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toEqual({ name: 'Técnica' });
    expect(request.request.withCredentials).toBe(true);
    request.flush(categoryResponse('category-id'));
    await create;

    const update = service.update('category-id', { name: 'Mobilidade' });
    request = http.expectOne('/api/categories/category-id');
    expect(request.request.method).toBe('PUT');
    expect(request.request.body).toEqual({ name: 'Mobilidade' });
    expect(request.request.withCredentials).toBe(true);
    request.flush(categoryResponse('category-id'));
    await update;

    const status = service.setStatus('category-id', false);
    request = http.expectOne('/api/categories/category-id/status');
    expect(request.request.method).toBe('PATCH');
    expect(request.request.body).toEqual({ active: false });
    expect(request.request.withCredentials).toBe(true);
    request.flush(categoryResponse('category-id'));
    await status;

    const remove = service.delete('category-id');
    request = http.expectOne('/api/categories/category-id');
    expect(request.request.method).toBe('DELETE');
    expect(request.request.withCredentials).toBe(true);
    request.flush(null, { status: 204, statusText: 'No Content' });
    await remove;
  });
});

function categoryResponse(id: string) {
  return {
    id,
    name: 'Técnica',
    active: true,
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
  };
}
