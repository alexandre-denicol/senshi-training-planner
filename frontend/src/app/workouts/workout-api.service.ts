import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { BlockCategory, BlockSequenceItem } from '../blocks/block-api.service';

export interface WorkoutListItem {
  id: string;
  name: string;
  active: boolean;
  blockCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface WorkoutBlock {
  id: string;
  name: string;
  description: string | null;
  sequence: BlockSequenceItem[];
  active: boolean;
  position: number;
  category: BlockCategory;
}

export interface WorkoutDetail {
  id: string;
  name: string;
  active: boolean;
  blocks: WorkoutBlock[];
  createdAt: string;
  updatedAt: string;
}

export interface WorkoutRequest {
  name: string;
  blockIds: string[];
}

@Injectable({ providedIn: 'root' })
export class WorkoutApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api/workouts';

  list(): Promise<WorkoutListItem[]> {
    return firstValueFrom(
      this.http.get<WorkoutListItem[]>(this.baseUrl, { withCredentials: true }),
    );
  }

  getById(id: string): Promise<WorkoutDetail> {
    return firstValueFrom(
      this.http.get<WorkoutDetail>(`${this.baseUrl}/${id}`, { withCredentials: true }),
    );
  }

  create(request: WorkoutRequest): Promise<WorkoutDetail> {
    return firstValueFrom(
      this.http.post<WorkoutDetail>(this.baseUrl, request, { withCredentials: true }),
    );
  }

  update(id: string, request: WorkoutRequest): Promise<WorkoutDetail> {
    return firstValueFrom(
      this.http.put<WorkoutDetail>(`${this.baseUrl}/${id}`, request, { withCredentials: true }),
    );
  }

  setStatus(id: string, active: boolean): Promise<WorkoutListItem> {
    return firstValueFrom(
      this.http.patch<WorkoutListItem>(`${this.baseUrl}/${id}/status`, { active }, { withCredentials: true }),
    );
  }

  delete(id: string): Promise<void> {
    return firstValueFrom(
      this.http.delete<void>(`${this.baseUrl}/${id}`, { withCredentials: true }),
    );
  }
}
