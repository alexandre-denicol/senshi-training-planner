import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { firstValueFrom } from 'rxjs';

export interface BlockCategory {
  id: string;
  name: string;
}

export interface Block {
  id: string;
  name: string;
  active: boolean;
  category: BlockCategory;
  createdAt: string;
  updatedAt: string;
}

export interface BlockRequest {
  name: string;
  categoryId: string;
}

@Injectable({ providedIn: 'root' })
export class BlockApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api/blocks';

  list(): Promise<Block[]> {
    return firstValueFrom(
      this.http.get<Block[]>(this.baseUrl, { withCredentials: true }),
    );
  }

  create(request: BlockRequest): Promise<Block> {
    return firstValueFrom(
      this.http.post<Block>(this.baseUrl, request, { withCredentials: true }),
    );
  }

  update(id: string, request: BlockRequest): Promise<Block> {
    return firstValueFrom(
      this.http.put<Block>(`${this.baseUrl}/${id}`, request, { withCredentials: true }),
    );
  }

  setStatus(id: string, active: boolean): Promise<Block> {
    return firstValueFrom(
      this.http.patch<Block>(`${this.baseUrl}/${id}/status`, { active }, { withCredentials: true }),
    );
  }

  delete(id: string): Promise<void> {
    return firstValueFrom(
      this.http.delete<void>(`${this.baseUrl}/${id}`, { withCredentials: true }),
    );
  }
}
