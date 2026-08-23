import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { WorkoutApiService } from './workout-api.service';

describe('WorkoutApiService', () => {
  let service: WorkoutApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });

    service = TestBed.inject(WorkoutApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    http.verify();
  });

  it('uses expected workout API endpoints with credentials', async () => {
    const list = service.list();
    let request = http.expectOne('/api/workouts');
    expect(request.request.method).toBe('GET');
    expect(request.request.withCredentials).toBe(true);
    request.flush([]);
    await list;

    const detail = service.getById('workout-id');
    request = http.expectOne('/api/workouts/workout-id');
    expect(request.request.method).toBe('GET');
    expect(request.request.withCredentials).toBe(true);
    request.flush(workoutDetail('workout-id'));
    await detail;

    const create = service.create({ name: 'Treino', blockIds: ['block-1', 'block-2'] });
    request = http.expectOne('/api/workouts');
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toEqual({ name: 'Treino', blockIds: ['block-1', 'block-2'] });
    expect(request.request.withCredentials).toBe(true);
    request.flush(workoutDetail('workout-id'));
    await create;

    const update = service.update('workout-id', { name: 'Editado', blockIds: ['block-2', 'block-1'] });
    request = http.expectOne('/api/workouts/workout-id');
    expect(request.request.method).toBe('PUT');
    expect(request.request.body).toEqual({ name: 'Editado', blockIds: ['block-2', 'block-1'] });
    expect(request.request.withCredentials).toBe(true);
    request.flush(workoutDetail('workout-id'));
    await update;

    const status = service.setStatus('workout-id', false);
    request = http.expectOne('/api/workouts/workout-id/status');
    expect(request.request.method).toBe('PATCH');
    expect(request.request.body).toEqual({ active: false });
    expect(request.request.withCredentials).toBe(true);
    request.flush(workoutListItem('workout-id'));
    await status;

    const remove = service.delete('workout-id');
    request = http.expectOne('/api/workouts/workout-id');
    expect(request.request.method).toBe('DELETE');
    expect(request.request.withCredentials).toBe(true);
    request.flush(null, { status: 204, statusText: 'No Content' });
    await remove;
  });
});

function workoutListItem(id: string) {
  return {
    id,
    name: 'Treino',
    active: true,
    blockCount: 2,
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
  };
}

function workoutDetail(id: string) {
  return {
    ...workoutListItem(id),
    blocks: [
      {
        id: 'block-1',
        name: 'Bloco',
        active: true,
        position: 1,
        category: { id: 'category-id', name: 'Categoria' },
      },
    ],
  };
}
