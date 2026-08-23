import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { ScheduleApiService } from './schedule-api.service';

describe('ScheduleApiService', () => {
  let service: ScheduleApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });

    service = TestBed.inject(ScheduleApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    http.verify();
  });

  it('uses expected schedule API endpoints with credentials', async () => {
    const list = service.list('2026-08-01', '2026-08-31');
    let request = http.expectOne('/api/schedule?from=2026-08-01&to=2026-08-31');
    expect(request.request.method).toBe('GET');
    expect(request.request.withCredentials).toBe(true);
    request.flush([]);
    await list;

    const create = service.create({ workoutId: 'workout-id', scheduledDate: '2026-08-24' });
    request = http.expectOne('/api/schedule');
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toEqual({ workoutId: 'workout-id', scheduledDate: '2026-08-24' });
    expect(request.request.withCredentials).toBe(true);
    request.flush(entry('schedule-id'));
    await create;

    const update = service.update('schedule-id', { workoutId: 'other-workout', scheduledDate: '2026-08-25' });
    request = http.expectOne('/api/schedule/schedule-id');
    expect(request.request.method).toBe('PUT');
    expect(request.request.body).toEqual({ workoutId: 'other-workout', scheduledDate: '2026-08-25' });
    expect(request.request.withCredentials).toBe(true);
    request.flush(entry('schedule-id'));
    await update;

    const remove = service.delete('schedule-id');
    request = http.expectOne('/api/schedule/schedule-id');
    expect(request.request.method).toBe('DELETE');
    expect(request.request.withCredentials).toBe(true);
    request.flush(null, { status: 204, statusText: 'No Content' });
    await remove;

    const complete = service.complete('schedule-id', { participantCount: 12, participantNames: ['João'], notes: 'Boa resposta.' });
    request = http.expectOne('/api/schedule/schedule-id/complete');
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toEqual({ participantCount: 12, participantNames: ['João'], notes: 'Boa resposta.' });
    expect(request.request.withCredentials).toBe(true);
    request.flush({});
    await complete;
  });
});

function entry(id: string) {
  return {
    id,
    scheduledDate: '2026-08-24',
    workout: { id: 'workout-id', name: 'Treino', active: true },
    createdAt: '2026-08-23T00:00:00Z',
    updatedAt: '2026-08-23T00:00:00Z',
  };
}
