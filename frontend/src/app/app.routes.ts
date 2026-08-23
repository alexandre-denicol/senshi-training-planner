import { Routes } from '@angular/router';
import { adminGuard, authGuard, loginGuard } from './auth/auth.guards';
import { CategoriesPage } from './categories/categories-page';
import { LoginPage } from './login/login.page';
import { ProfessorsPage } from './professors/professors-page';
import { AppShell } from './shell/app-shell';
import { WelcomePage } from './welcome/welcome-page';

export const routes: Routes = [
  { path: 'login', component: LoginPage, canActivate: [loginGuard] },
  {
    path: 'app',
    component: AppShell,
    canActivate: [authGuard],
    children: [
      { path: '', pathMatch: 'full', component: WelcomePage },
      { path: 'categorias', component: CategoriesPage, canActivate: [adminGuard] },
      { path: 'professores', component: ProfessorsPage, canActivate: [adminGuard] },
    ],
  },
  { path: '', pathMatch: 'full', redirectTo: 'app' },
  { path: '**', redirectTo: 'app' },
];
