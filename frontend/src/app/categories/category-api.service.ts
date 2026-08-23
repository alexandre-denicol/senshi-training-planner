import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { firstValueFrom } from 'rxjs';

export interface Category {
  id: string;
  name: string;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CategoryNameRequest {
  name: string;
}

@Injectable({ providedIn: 'root' })
export class CategoryApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api/categories';

  list(): Promise<Category[]> {
    return firstValueFrom(
      this.http.get<Category[]>(this.baseUrl, { withCredentials: true }),
    );
  }

  create(request: CategoryNameRequest): Promise<Category> {
    return firstValueFrom(
      this.http.post<Category>(this.baseUrl, request, { withCredentials: true }),
    );
  }

  update(id: string, request: CategoryNameRequest): Promise<Category> {
    return firstValueFrom(
      this.http.put<Category>(`${this.baseUrl}/${id}`, request, { withCredentials: true }),
    );
  }

  setStatus(id: string, active: boolean): Promise<Category> {
    return firstValueFrom(
      this.http.patch<Category>(`${this.baseUrl}/${id}/status`, { active }, { withCredentials: true }),
    );
  }

  delete(id: string): Promise<void> {
    return firstValueFrom(
      this.http.delete<void>(`${this.baseUrl}/${id}`, { withCredentials: true }),
    );
  }
}
