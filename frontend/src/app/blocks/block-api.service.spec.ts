import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { BlockApiService } from './block-api.service';

describe('BlockApiService', () => {
  let service: BlockApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });

    service = TestBed.inject(BlockApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    http.verify();
  });

  it('uses expected block API endpoints with credentials', async () => {
    const list = service.list();
    let request = http.expectOne('/api/blocks');
    expect(request.request.method).toBe('GET');
    expect(request.request.withCredentials).toBe(true);
    request.flush([]);
    await list;

    const create = service.create({ name: 'Base', categoryId: 'category-id' });
    request = http.expectOne('/api/blocks');
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toEqual({ name: 'Base', categoryId: 'category-id' });
    expect(request.request.withCredentials).toBe(true);
    request.flush(blockResponse('block-id'));
    await create;

    const update = service.update('block-id', { name: 'Avançado', categoryId: 'other-category-id' });
    request = http.expectOne('/api/blocks/block-id');
    expect(request.request.method).toBe('PUT');
    expect(request.request.body).toEqual({ name: 'Avançado', categoryId: 'other-category-id' });
    expect(request.request.withCredentials).toBe(true);
    request.flush(blockResponse('block-id'));
    await update;

    const status = service.setStatus('block-id', false);
    request = http.expectOne('/api/blocks/block-id/status');
    expect(request.request.method).toBe('PATCH');
    expect(request.request.body).toEqual({ active: false });
    expect(request.request.withCredentials).toBe(true);
    request.flush(blockResponse('block-id'));
    await status;

    const remove = service.delete('block-id');
    request = http.expectOne('/api/blocks/block-id');
    expect(request.request.method).toBe('DELETE');
    expect(request.request.withCredentials).toBe(true);
    request.flush(null, { status: 204, statusText: 'No Content' });
    await remove;
  });
});

function blockResponse(id: string) {
  return {
    id,
    name: 'Base',
    active: true,
    category: {
      id: 'category-id',
      name: 'Categoria',
    },
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
  };
}
