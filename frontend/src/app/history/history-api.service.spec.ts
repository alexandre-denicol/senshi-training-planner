import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { HistoryApiService } from './history-api.service';

describe('HistoryApiService', () => {
  let service: HistoryApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });

    service = TestBed.inject(HistoryApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    http.verify();
  });

  it('uses expected history API endpoints with credentials', async () => {
    const list = service.list('2026-08-01', '2026-08-31');
    let request = http.expectOne('/api/history?from=2026-08-01&to=2026-08-31');
    expect(request.request.method).toBe('GET');
    expect(request.request.withCredentials).toBe(true);
    request.flush([]);
    await list;

    const detail = service.getById('history-id');
    request = http.expectOne('/api/history/history-id');
    expect(request.request.method).toBe('GET');
    expect(request.request.withCredentials).toBe(true);
    request.flush({
      id: 'history-id',
      trainingDate: '2026-08-24',
      workoutName: 'Treino',
      participantCount: 12,
      participantNames: ['João'],
      notes: 'Boa resposta.',
      completedByName: 'Professor',
      completedAt: '2026-08-24T15:32:00Z',
      blocks: [],
    });
    await detail;
  });
});
