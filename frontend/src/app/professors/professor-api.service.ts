import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { firstValueFrom } from 'rxjs';

export interface Professor {
  id: string;
  name: string;
  email: string;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreateProfessorRequest {
  name: string;
  email: string;
  password: string;
}

export interface UpdateProfessorRequest {
  name: string;
  email: string;
}

@Injectable({ providedIn: 'root' })
export class ProfessorApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api/professors';

  list(): Promise<Professor[]> {
    return firstValueFrom(
      this.http.get<Professor[]>(this.baseUrl, { withCredentials: true }),
    );
  }

  create(request: CreateProfessorRequest): Promise<Professor> {
    return firstValueFrom(
      this.http.post<Professor>(this.baseUrl, request, { withCredentials: true }),
    );
  }

  update(id: string, request: UpdateProfessorRequest): Promise<Professor> {
    return firstValueFrom(
      this.http.put<Professor>(`${this.baseUrl}/${id}`, request, { withCredentials: true }),
    );
  }

  setStatus(id: string, active: boolean): Promise<Professor> {
    return firstValueFrom(
      this.http.patch<Professor>(`${this.baseUrl}/${id}/status`, { active }, { withCredentials: true }),
    );
  }

  setPassword(id: string, password: string): Promise<void> {
    return firstValueFrom(
      this.http.put<void>(`${this.baseUrl}/${id}/password`, { password }, { withCredentials: true }),
    );
  }

  delete(id: string): Promise<void> {
    return firstValueFrom(
      this.http.delete<void>(`${this.baseUrl}/${id}`, { withCredentials: true }),
    );
  }
}
