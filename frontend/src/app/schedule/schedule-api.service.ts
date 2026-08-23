import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { firstValueFrom } from 'rxjs';

export interface ScheduleWorkout {
  id: string;
  name: string;
  active: boolean;
}

export interface ScheduleEntry {
  id: string;
  scheduledDate: string;
  workout: ScheduleWorkout;
  createdAt: string;
  updatedAt: string;
}

export interface ScheduleRequest {
  workoutId: string;
  scheduledDate: string;
}

@Injectable({ providedIn: 'root' })
export class ScheduleApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api/schedule';

  list(from: string, to: string): Promise<ScheduleEntry[]> {
    const params = new HttpParams().set('from', from).set('to', to);
    return firstValueFrom(
      this.http.get<ScheduleEntry[]>(this.baseUrl, { params, withCredentials: true }),
    );
  }

  create(request: ScheduleRequest): Promise<ScheduleEntry> {
    return firstValueFrom(
      this.http.post<ScheduleEntry>(this.baseUrl, request, { withCredentials: true }),
    );
  }

  update(id: string, request: ScheduleRequest): Promise<ScheduleEntry> {
    return firstValueFrom(
      this.http.put<ScheduleEntry>(`${this.baseUrl}/${id}`, request, { withCredentials: true }),
    );
  }

  delete(id: string): Promise<void> {
    return firstValueFrom(
      this.http.delete<void>(`${this.baseUrl}/${id}`, { withCredentials: true }),
    );
  }
}
