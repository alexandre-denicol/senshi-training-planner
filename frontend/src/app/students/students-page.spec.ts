import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { StudentsPage } from './students-page';
import { Student } from './student-api.service';

describe('StudentsPage', () => {
  let component: StudentsPage;
  let fixture: ComponentFixture<StudentsPage>;
  let httpMock: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StudentsPage, RouterTestingModule, HttpClientTestingModule],
    }).compileComponents();

    fixture = TestBed.createComponent(StudentsPage);
    component = fixture.componentInstance;
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should load students on init', async () => {
    const mockStudents: Student[] = [
      {
        id: '1',
        name: 'Ana Silva',
        active: true,
        phone: '(48) 99999-0000',
        createdAt: '2026-01-01T00:00:00Z',
        updatedAt: '2026-01-01T00:00:00Z',
      },
    ];

    fixture.detectChanges();

    const req = httpMock.expectOne('/api/students');
    expect(req.request.method).toBe('GET');
    req.flush(mockStudents);

    await fixture.whenStable();

    expect(component['students']()).toEqual(mockStudents);
  });

  it('should display empty state when no students exist', async () => {
    fixture.detectChanges();
    const req = httpMock.expectOne('/api/students');
    req.flush([]);

    await fixture.whenStable();

    expect(component['students']().length).toBe(0);
    expect(component['hasStudents']()).toBe(false);
  });

  it('should display loading state initially', async () => {
    fixture.detectChanges();
    expect(component['loading']()).toBe(true);

    const req = httpMock.expectOne('/api/students');
    req.flush([]);

    await fixture.whenStable();

    expect(component['loading']()).toBe(false);
  });

  it('should handle error loading students', async () => {
    fixture.detectChanges();
    const req = httpMock.expectOne('/api/students');
    req.error(new ErrorEvent('Network error'));

    await fixture.whenStable();

    expect(component['errorMessage']()).toContain('Não foi possível');
  });
});
