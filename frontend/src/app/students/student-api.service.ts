import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { firstValueFrom } from 'rxjs';

export interface Student {
  id: string;
  name: string;
  active: boolean;
  birthDate?: string;
  phone?: string;
  guardianName?: string;
  guardianPhone?: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

export interface StudentCreateRequest {
  name: string;
  birthDate?: string;
  phone?: string;
  guardianName?: string;
  guardianPhone?: string;
  notes?: string;
}

export interface StudentUpdateRequest {
  name: string;
  birthDate?: string;
  phone?: string;
  guardianName?: string;
  guardianPhone?: string;
  notes?: string;
}

@Injectable({ providedIn: 'root' })
export class StudentApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api/students';

  list(): Promise<Student[]> {
    return firstValueFrom(
      this.http.get<Student[]>(this.baseUrl, { withCredentials: true }),
    );
  }

  create(request: StudentCreateRequest): Promise<Student> {
    return firstValueFrom(
      this.http.post<Student>(this.baseUrl, request, { withCredentials: true }),
    );
  }

  update(id: string, request: StudentUpdateRequest): Promise<Student> {
    return firstValueFrom(
      this.http.put<Student>(`${this.baseUrl}/${id}`, request, { withCredentials: true }),
    );
  }

  setStatus(id: string, active: boolean): Promise<Student> {
    return firstValueFrom(
      this.http.patch<Student>(`${this.baseUrl}/${id}/status`, { active }, { withCredentials: true }),
    );
  }
}
