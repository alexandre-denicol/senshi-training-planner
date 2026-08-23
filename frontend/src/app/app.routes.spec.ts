import { routes } from './app.routes';
import { adminGuard, authGuard } from './auth/auth.guards';
import { HistoryPage } from './history/history-page';
import { SchedulePage } from './schedule/schedule-page';

describe('routes', () => {
  it('protects Agenda with authentication only so ADMIN and PROFESSOR can access it', () => {
    const appRoute = routes.find((route) => route.path === 'app');
    const agendaRoute = appRoute?.children?.find((route) => route.path === 'agenda');

    expect(appRoute?.canActivate).toEqual([authGuard]);
    expect(agendaRoute?.component).toBe(SchedulePage);
    expect(agendaRoute?.canActivate).toBeUndefined();
    expect(agendaRoute?.canActivate).not.toEqual([adminGuard]);
  });

  it('protects Histórico with authentication only so ADMIN and PROFESSOR can access it', () => {
    const appRoute = routes.find((route) => route.path === 'app');
    const historyRoute = appRoute?.children?.find((route) => route.path === 'historico');

    expect(appRoute?.canActivate).toEqual([authGuard]);
    expect(historyRoute?.component).toBe(HistoryPage);
    expect(historyRoute?.canActivate).toBeUndefined();
    expect(historyRoute?.canActivate).not.toEqual([adminGuard]);
  });
});
