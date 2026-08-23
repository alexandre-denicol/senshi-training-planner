import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { firstValueFrom } from 'rxjs';

export interface HistoryListItem {
  id: string;
  trainingDate: string;
  workoutName: string;
  blockCount: number;
  participantCount: number | null;
  completedByName: string;
  completedAt: string;
  scheduleEntryId: string;
}

export interface HistoryBlock {
  position: number;
  blockName: string;
  categoryName: string;
}

export interface HistoryDetail {
  id: string;
  trainingDate: string;
  workoutName: string;
  participantCount: number | null;
  participantNames: string[];
  completedByName: string;
  completedAt: string;
  blocks: HistoryBlock[];
}

@Injectable({ providedIn: 'root' })
export class HistoryApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api/history';

  list(from: string, to: string): Promise<HistoryListItem[]> {
    const params = new HttpParams().set('from', from).set('to', to);
    return firstValueFrom(
      this.http.get<HistoryListItem[]>(this.baseUrl, { params, withCredentials: true }),
    );
  }

  getById(id: string): Promise<HistoryDetail> {
    return firstValueFrom(
      this.http.get<HistoryDetail>(`${this.baseUrl}/${id}`, { withCredentials: true }),
    );
  }
}
