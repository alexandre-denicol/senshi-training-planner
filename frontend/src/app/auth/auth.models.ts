export type UserRole = 'ADMIN' | 'PROFESSOR';

export interface AuthUser {
  id: string;
  name: string;
  email: string;
  role: UserRole;
}

export type AuthStatus = 'checking' | 'authenticated' | 'unauthenticated';
