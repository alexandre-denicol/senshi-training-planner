import { Routes } from '@angular/router';
import { adminGuard, authGuard, loginGuard } from './auth/auth.guards';
import { BlocksPage } from './blocks/blocks-page';
import { CategoriesPage } from './categories/categories-page';
import { HistoryPage } from './history/history-page';
import { LoginPage } from './login/login.page';
import { SchedulePage } from './schedule/schedule-page';
import { ProfessorsPage } from './professors/professors-page';
import { AppShell } from './shell/app-shell';
import { WelcomePage } from './welcome/welcome-page';
import { WorkoutsPage } from './workouts/workouts-page';

export const routes: Routes = [
  { path: 'login', component: LoginPage, canActivate: [loginGuard] },
  {
    path: 'app',
    component: AppShell,
    canActivate: [authGuard],
    children: [
      { path: '', pathMatch: 'full', component: WelcomePage },
      { path: 'agenda', component: SchedulePage },
      { path: 'treinos', component: WorkoutsPage, canActivate: [adminGuard] },
      { path: 'blocos', component: BlocksPage, canActivate: [adminGuard] },
      { path: 'categorias', component: CategoriesPage, canActivate: [adminGuard] },
      { path: 'historico', component: HistoryPage },
      { path: 'professores', component: ProfessorsPage, canActivate: [adminGuard] },
    ],
  },
  { path: '', pathMatch: 'full', redirectTo: 'app' },
  { path: '**', redirectTo: 'app' },
];
