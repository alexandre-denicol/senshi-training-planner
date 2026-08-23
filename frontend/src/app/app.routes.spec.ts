import { routes } from './app.routes';
import { adminGuard, authGuard } from './auth/auth.guards';
import { BlocksPage } from './blocks/blocks-page';
import { CategoriesPage } from './categories/categories-page';
import { HistoryPage } from './history/history-page';
import { ProfessorsPage } from './professors/professors-page';
import { SchedulePage } from './schedule/schedule-page';
import { WorkoutsPage } from './workouts/workouts-page';

describe('routes', () => {
  it('protects Agenda with authentication only so ADMIN and PROFESSOR can access it', () => {
    const appRoute = routes.find((route) => route.path === 'app');
    const agendaRoute = appRoute?.children?.find((route) => route.path === 'agenda');

    expect(appRoute?.canActivate).toEqual([authGuard]);
    expect(agendaRoute?.component).toBe(SchedulePage);
    expect(agendaRoute?.canActivate).toBeUndefined();
    expect(agendaRoute?.canActivate).not.toEqual([adminGuard]);
  });

  it('protects training catalog routes with authentication only so ADMIN and PROFESSOR can operate them', () => {
    const appRoute = routes.find((route) => route.path === 'app');
    const categoriesRoute = appRoute?.children?.find((route) => route.path === 'categorias');
    const blocksRoute = appRoute?.children?.find((route) => route.path === 'blocos');
    const workoutsRoute = appRoute?.children?.find((route) => route.path === 'treinos');

    expect(appRoute?.canActivate).toEqual([authGuard]);
    expect(categoriesRoute?.component).toBe(CategoriesPage);
    expect(blocksRoute?.component).toBe(BlocksPage);
    expect(workoutsRoute?.component).toBe(WorkoutsPage);
    expect(categoriesRoute?.canActivate).toBeUndefined();
    expect(blocksRoute?.canActivate).toBeUndefined();
    expect(workoutsRoute?.canActivate).toBeUndefined();
  });

  it('protects Histórico with authentication only so ADMIN and PROFESSOR can access it', () => {
    const appRoute = routes.find((route) => route.path === 'app');
    const historyRoute = appRoute?.children?.find((route) => route.path === 'historico');

    expect(appRoute?.canActivate).toEqual([authGuard]);
    expect(historyRoute?.component).toBe(HistoryPage);
    expect(historyRoute?.canActivate).toBeUndefined();
    expect(historyRoute?.canActivate).not.toEqual([adminGuard]);
  });

  it('keeps Professores route ADMIN-only', () => {
    const appRoute = routes.find((route) => route.path === 'app');
    const professorsRoute = appRoute?.children?.find((route) => route.path === 'professores');

    expect(professorsRoute?.component).toBe(ProfessorsPage);
    expect(professorsRoute?.canActivate).toEqual([adminGuard]);
  });
});
