import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Student, StudentApiService } from './student-api.service';

describe('StudentApiService', () => {
  let service: StudentApiService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [StudentApiService],
    });

    service = TestBed.inject(StudentApiService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should list students', async () => {
    const mockStudents: Student[] = [
      {
        id: '1',
        name: 'Ana Silva',
        active: true,
        createdAt: '2026-01-01T00:00:00Z',
        updatedAt: '2026-01-01T00:00:00Z',
      },
      {
        id: '2',
        name: 'João Santos',
        active: true,
        phone: '(48) 99999-1111',
        createdAt: '2026-01-02T00:00:00Z',
        updatedAt: '2026-01-02T00:00:00Z',
      },
    ];

    const promise = service.list();

    const req = httpMock.expectOne('/api/students');
    expect(req.request.method).toBe('GET');
    expect(req.request.withCredentials).toBe(true);

    req.flush(mockStudents);

    const result = await promise;
    expect(result).toEqual(mockStudents);
  });

  it('should create a student', async () => {
    const newStudent = {
      name: 'Maria Oliveira',
      birthDate: '2010-05-20',
      phone: '(48) 98888-2222',
    };

    const createdStudent: Student = {
      id: '123',
      ...newStudent,
      active: true,
      createdAt: '2026-01-03T00:00:00Z',
      updatedAt: '2026-01-03T00:00:00Z',
    };

    const promise = service.create(newStudent);

    const req = httpMock.expectOne('/api/students');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(newStudent);
    expect(req.request.withCredentials).toBe(true);

    req.flush(createdStudent);

    const result = await promise;
    expect(result).toEqual(createdStudent);
  });

  it('should update a student', async () => {
    const updatedData = {
      name: 'Ana Oliveira',
      phone: '(48) 99999-0000',
    };

    const updatedStudent: Student = {
      id: '1',
      name: 'Ana Oliveira',
      active: true,
      phone: '(48) 99999-0000',
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-04T00:00:00Z',
    };

    const promise = service.update('1', updatedData);

    const req = httpMock.expectOne('/api/students/1');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual(updatedData);
    expect(req.request.withCredentials).toBe(true);

    req.flush(updatedStudent);

    const result = await promise;
    expect(result).toEqual(updatedStudent);
  });

  it('should set student status', async () => {
    const deactivatedStudent: Student = {
      id: '1',
      name: 'Ana Silva',
      active: false,
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-05T00:00:00Z',
    };

    const promise = service.setStatus('1', false);

    const req = httpMock.expectOne('/api/students/1/status');
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ active: false });
    expect(req.request.withCredentials).toBe(true);

    req.flush(deactivatedStudent);

    const result = await promise;
    expect(result).toEqual(deactivatedStudent);
  });

  it('should handle optional fields in create', async () => {
    const newStudent = {
      name: 'Carlos Silva',
    };

    const createdStudent: Student = {
      id: '456',
      name: 'Carlos Silva',
      active: true,
      createdAt: '2026-01-06T00:00:00Z',
      updatedAt: '2026-01-06T00:00:00Z',
    };

    const promise = service.create(newStudent);

    const req = httpMock.expectOne('/api/students');
    req.flush(createdStudent);

    const result = await promise;
    expect(result.birthDate).toBeUndefined();
    expect(result.phone).toBeUndefined();
  });
});
