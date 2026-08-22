import { Routes } from '@angular/router';
import { authGuard, loginGuard } from './auth/auth.guards';
import { LoginPage } from './login/login.page';
import { AppShell } from './shell/app-shell';

export const routes: Routes = [
  { path: 'login', component: LoginPage, canActivate: [loginGuard] },
  { path: 'app', component: AppShell, canActivate: [authGuard] },
  { path: '', pathMatch: 'full', redirectTo: 'app' },
  { path: '**', redirectTo: 'app' },
];
