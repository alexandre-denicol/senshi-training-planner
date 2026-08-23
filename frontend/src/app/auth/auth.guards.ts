import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from './auth.service';

export const authGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const user = await auth.restoreSession();

  return user ? true : router.createUrlTree(['/login']);
};

export const loginGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const user = await auth.restoreSession();

  return user ? router.createUrlTree(['/app']) : true;
};

export const adminGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const user = await auth.restoreSession();

  if (!user) {
    return router.createUrlTree(['/login']);
  }

  return user.role === 'ADMIN' ? true : router.createUrlTree(['/app']);
};
