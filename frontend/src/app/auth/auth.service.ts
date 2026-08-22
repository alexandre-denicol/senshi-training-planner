import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { AuthStatus, AuthUser } from './auth.models';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly userSignal = signal<AuthUser | null>(null);
  private readonly statusSignal = signal<AuthStatus>('checking');
  private restorePromise: Promise<AuthUser | null> | null = null;

  readonly currentUser = this.userSignal.asReadonly();
  readonly status = this.statusSignal.asReadonly();
  readonly isAuthenticated = computed(() => this.statusSignal() === 'authenticated');

  restoreSession(): Promise<AuthUser | null> {
    if (this.restorePromise) {
      return this.restorePromise;
    }

    this.statusSignal.set('checking');
    this.restorePromise = firstValueFrom(
      this.http.get<AuthUser>('/api/auth/me', { withCredentials: true }),
    )
      .then((user) => {
        this.setAuthenticated(user);
        return user;
      })
      .catch((error: unknown) => {
        if (error instanceof HttpErrorResponse && error.status === 401) {
          this.setUnauthenticated();
          return null;
        }

        this.setUnauthenticated();
        return null;
      })
      .finally(() => {
        this.restorePromise = null;
      });

    return this.restorePromise;
  }

  async login(email: string, password: string): Promise<AuthUser> {
    try {
      const user = await firstValueFrom(
        this.http.post<AuthUser>(
          '/api/auth/login',
          { email, password },
          { withCredentials: true },
        ),
      );

      this.setAuthenticated(user);
      return user;
    } catch (error) {
      this.setUnauthenticated();
      throw error;
    }
  }

  async logout(): Promise<void> {
    try {
      await firstValueFrom(
        this.http.post<void>('/api/auth/logout', {}, { withCredentials: true }),
      );
    } finally {
      this.setUnauthenticated();
    }
  }

  private setAuthenticated(user: AuthUser): void {
    this.userSignal.set(user);
    this.statusSignal.set('authenticated');
  }

  private setUnauthenticated(): void {
    this.userSignal.set(null);
    this.statusSignal.set('unauthenticated');
  }
}
