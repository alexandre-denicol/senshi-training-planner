import { Routes } from '@angular/router';
import { adminGuard, authGuard, loginGuard } from './auth/auth.guards';
import { BlocksPage } from './blocks/blocks-page';
import { CategoriesPage } from './categories/categories-page';
import { DashboardPage } from './dashboard/dashboard-page';
import { HistoryPage } from './history/history-page';
import { LoginPage } from './login/login.page';
import { SchedulePage } from './schedule/schedule-page';
import { ProfessorsPage } from './professors/professors-page';
import { AppShell } from './shell/app-shell';
import { WorkoutsPage } from './workouts/workouts-page';

export const routes: Routes = [
  { path: 'login', component: LoginPage, canActivate: [loginGuard] },
  {
    path: 'app',
    component: AppShell,
    canActivate: [authGuard],
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'dashboard' },
      { path: 'dashboard', component: DashboardPage },
      { path: 'agenda', component: SchedulePage },
      { path: 'treinos', component: WorkoutsPage },
      { path: 'blocos', component: BlocksPage },
      { path: 'categorias', component: CategoriesPage },
      { path: 'historico', component: HistoryPage },
      { path: 'professores', component: ProfessorsPage, canActivate: [adminGuard] },
    ],
  },
  { path: '', pathMatch: 'full', redirectTo: 'app' },
  { path: '**', redirectTo: 'app' },
];
