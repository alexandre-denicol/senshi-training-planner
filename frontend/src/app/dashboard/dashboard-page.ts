import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { AuthService } from '../auth/auth.service';
import { BlockApiService } from '../blocks/block-api.service';
import { CategoryApiService } from '../categories/category-api.service';
import { HistoryApiService, HistoryDetail, HistoryListItem } from '../history/history-api.service';
import { ScheduleApiService, ScheduleEntry } from '../schedule/schedule-api.service';
import { WorkoutApiService, WorkoutDetail } from '../workouts/workout-api.service';

interface SummaryCard {
  label: string;
  value: number;
  icon: string;
  link: string;
  action: string;
}

@Component({
  imports: [ButtonModule, CommonModule, DialogModule, RouterLink],
  selector: 'app-dashboard-page',
  styleUrl: './dashboard-page.css',
  templateUrl: './dashboard-page.html',
})
export class DashboardPage implements OnInit {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly scheduleApi = inject(ScheduleApiService);
  private readonly workoutApi = inject(WorkoutApiService);
  private readonly blockApi = inject(BlockApiService);
  private readonly categoryApi = inject(CategoryApiService);
  private readonly historyApi = inject(HistoryApiService);
  private readonly dateFormatter = new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: 'short' });
  private readonly fullDateFormatter = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short' });

  protected readonly loading = signal(true);
  protected readonly errorMessage = signal('');
  protected readonly scheduledEntries = signal<ScheduleEntry[]>([]);
  protected readonly workoutCount = signal(0);
  protected readonly blockCount = signal(0);
  protected readonly categoryCount = signal(0);
  protected readonly historyItems = signal<HistoryListItem[]>([]);
  protected readonly workoutDetail = signal<WorkoutDetail | null>(null);
  protected readonly workoutDetailLoading = signal(false);
  protected readonly workoutDetailError = signal('');
  protected readonly historyDetail = signal<HistoryDetail | null>(null);
  protected readonly historyDetailLoading = signal(false);
  protected readonly historyDetailError = signal('');
  protected readonly userName = computed(() => this.auth.currentUser()?.name ?? 'Professor');
  protected readonly upcomingEntries = computed(() =>
    this.scheduledEntries()
      .filter((entry) => !entry.completedAt)
      .sort(compareScheduleEntries)
      .slice(0, 3),
  );
  protected readonly recentHistory = computed(() =>
    this.historyItems()
      .slice()
      .sort(compareHistoryItems)
      .slice(0, 3),
  );
  protected readonly summaryCards = computed<SummaryCard[]>(() => [
    {
      label: 'Treinos agendados',
      value: this.scheduledEntries().filter((entry) => !entry.completedAt).length,
      icon: 'pi pi-calendar',
      link: '/app/agenda',
      action: 'Ver agenda',
    },
    {
      label: 'Total de treinos',
      value: this.workoutCount(),
      icon: 'pi pi-list-check',
      link: '/app/treinos',
      action: 'Ver treinos',
    },
    {
      label: 'Total de blocos',
      value: this.blockCount(),
      icon: 'pi pi-th-large',
      link: '/app/blocos',
      action: 'Ver blocos',
    },
    {
      label: 'Total de categorias',
      value: this.categoryCount(),
      icon: 'pi pi-tags',
      link: '/app/categorias',
      action: 'Ver categorias',
    },
  ]);

  protected workoutDetailDialogOpen = false;
  protected historyDetailDialogOpen = false;
  protected selectedScheduleEntry: ScheduleEntry | null = null;

  async ngOnInit(): Promise<void> {
    await this.loadDashboard();
  }

  protected async loadDashboard(): Promise<void> {
    this.loading.set(true);
    this.errorMessage.set('');

    try {
      const today = todayISODate();
      const upcomingEnd = offsetISODate(new Date(), 92);
      const historyStart = offsetISODate(new Date(), -92);
      const [scheduled, workouts, blocks, categories, history] = await Promise.all([
        this.scheduleApi.list(today, upcomingEnd),
        this.workoutApi.list(),
        this.blockApi.list(),
        this.categoryApi.list(),
        this.historyApi.list(historyStart, today),
      ]);

      this.scheduledEntries.set(scheduled);
      this.workoutCount.set(workouts.length);
      this.blockCount.set(blocks.length);
      this.categoryCount.set(categories.length);
      this.historyItems.set(history);
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.loading.set(false);
    }
  }

  protected async openWorkoutDetail(entry: ScheduleEntry): Promise<void> {
    this.selectedScheduleEntry = entry;
    this.workoutDetail.set(null);
    this.workoutDetailError.set('');
    this.workoutDetailLoading.set(true);
    this.workoutDetailDialogOpen = true;

    try {
      this.workoutDetail.set(await this.workoutApi.getById(entry.workout.id));
    } catch (error) {
      this.workoutDetailError.set(error instanceof HttpErrorResponse && error.status === 404
        ? 'Treino não encontrado.'
        : 'Não foi possível carregar os detalhes do treino.');
    } finally {
      this.workoutDetailLoading.set(false);
    }
  }

  protected closeWorkoutDetail(): void {
    this.workoutDetailDialogOpen = false;
    this.selectedScheduleEntry = null;
    this.workoutDetail.set(null);
    this.workoutDetailError.set('');
  }

  protected async openHistoryDetail(item: HistoryListItem): Promise<void> {
    this.historyDetail.set(null);
    this.historyDetailError.set('');
    this.historyDetailLoading.set(true);
    this.historyDetailDialogOpen = true;

    try {
      this.historyDetail.set(await this.historyApi.getById(item.id));
    } catch (error) {
      this.historyDetailError.set(error instanceof HttpErrorResponse && error.status === 404
        ? 'Registro de histórico não encontrado.'
        : 'Não foi possível carregar o histórico.');
    } finally {
      this.historyDetailLoading.set(false);
    }
  }

  protected closeHistoryDetail(): void {
    this.historyDetailDialogOpen = false;
    this.historyDetail.set(null);
    this.historyDetailError.set('');
  }

  protected formatShortDate(value: string): string {
    const parts = parseISODateParts(value);
    if (!parts) {
      return '-';
    }

    return this.dateFormatter.format(new Date(parts.year, parts.month - 1, parts.day)).replace('.', '');
  }

  protected formatDay(value: string): string {
    const parts = parseISODateParts(value);
    return parts ? pad2(parts.day) : '--';
  }

  protected formatMonth(value: string): string {
    const parts = parseISODateParts(value);
    if (!parts) {
      return '';
    }

    return new Intl.DateTimeFormat('pt-BR', { month: 'short' })
      .format(new Date(parts.year, parts.month - 1, parts.day))
      .replace('.', '');
  }

  protected formatDate(value: string): string {
    const parts = parseISODateParts(value);
    if (!parts) {
      return '-';
    }

    return this.fullDateFormatter.format(new Date(parts.year, parts.month - 1, parts.day));
  }

  protected participantCountLabel(count: number | null | undefined): string {
    if (count === null || count === undefined) {
      return 'Participantes não informados';
    }

    return count === 1 ? '1 participante' : `${count} participantes`;
  }

  private async handleRequestError(error: unknown): Promise<void> {
    if (error instanceof HttpErrorResponse) {
      if (error.status === 401) {
        await this.router.navigateByUrl('/login');
        return;
      }
      if (error.status === 403) {
        await this.router.navigateByUrl('/app');
        return;
      }
    }

    this.errorMessage.set('Não foi possível carregar o dashboard. Tente novamente.');
  }
}

function compareScheduleEntries(first: ScheduleEntry, second: ScheduleEntry): number {
  if (first.scheduledDate !== second.scheduledDate) {
    return first.scheduledDate.localeCompare(second.scheduledDate);
  }

  return first.workout.name.localeCompare(second.workout.name, 'pt-BR', { sensitivity: 'base' });
}

function compareHistoryItems(first: HistoryListItem, second: HistoryListItem): number {
  if (first.trainingDate !== second.trainingDate) {
    return second.trainingDate.localeCompare(first.trainingDate);
  }

  return second.completedAt.localeCompare(first.completedAt);
}

function todayISODate(): string {
  return dateToISODate(new Date());
}

function offsetISODate(date: Date, days: number): string {
  return dateToISODate(new Date(date.getFullYear(), date.getMonth(), date.getDate() + days));
}

function dateToISODate(date: Date): string {
  return `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())}`;
}

function parseISODateParts(value: string): { year: number; month: number; day: number } | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) {
    return null;
  }

  return {
    year: Number(match[1]),
    month: Number(match[2]),
    day: Number(match[3]),
  };
}

function pad2(value: number): string {
  return value.toString().padStart(2, '0');
}
