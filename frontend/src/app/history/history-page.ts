import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { HistoryApiService, HistoryDetail, HistoryListItem } from './history-api.service';

interface HistoryGroup {
  date: string;
  label: string;
  items: HistoryListItem[];
}

@Component({
  imports: [ButtonModule, CommonModule, DialogModule],
  selector: 'app-history-page',
  styleUrl: './history-page.css',
  templateUrl: './history-page.html',
})
export class HistoryPage implements OnInit {
  private readonly api = inject(HistoryApiService);
  private readonly router = inject(Router);
  private readonly monthFormatter = new Intl.DateTimeFormat('pt-BR', { month: 'long', year: 'numeric' });
  private readonly groupFormatter = new Intl.DateTimeFormat('pt-BR', { weekday: 'long', day: '2-digit', month: 'long' });
  private readonly dateTimeFormatter = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short', timeStyle: 'short' });
  private readonly timeFormatter = new Intl.DateTimeFormat('pt-BR', { timeStyle: 'short' });

  protected readonly items = signal<HistoryListItem[]>([]);
  protected readonly currentMonth = signal(startOfMonth(new Date()));
  protected readonly loading = signal(true);
  protected readonly detailLoading = signal(false);
  protected readonly errorMessage = signal('');
  protected readonly selectedDetail = signal<HistoryDetail | null>(null);
  protected readonly hasItems = computed(() => this.items().length > 0);
  protected readonly periodLabel = computed(() => capitalize(this.monthFormatter.format(this.currentMonth())));
  protected readonly groups = computed(() => groupHistory(this.items(), this.groupFormatter));

  protected detailDialogOpen = false;

  async ngOnInit(): Promise<void> {
    await this.loadHistory();
  }

  protected async loadHistory(): Promise<void> {
    this.loading.set(true);
    this.errorMessage.set('');

    try {
      const range = monthRange(this.currentMonth());
      this.items.set(await this.api.list(range.from, range.to));
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.loading.set(false);
    }
  }

  protected async previousMonth(): Promise<void> {
    this.currentMonth.update((date) => startOfMonth(new Date(date.getFullYear(), date.getMonth() - 1, 1)));
    await this.loadHistory();
  }

  protected async nextMonth(): Promise<void> {
    this.currentMonth.update((date) => startOfMonth(new Date(date.getFullYear(), date.getMonth() + 1, 1)));
    await this.loadHistory();
  }

  protected async goToToday(): Promise<void> {
    this.currentMonth.set(startOfMonth(new Date()));
    await this.loadHistory();
  }

  protected async openDetail(item: HistoryListItem): Promise<void> {
    this.errorMessage.set('');
    this.detailLoading.set(true);
    this.detailDialogOpen = true;
    this.selectedDetail.set(null);

    try {
      this.selectedDetail.set(await this.api.getById(item.id));
    } catch (error) {
      this.detailDialogOpen = false;
      await this.handleRequestError(error);
    } finally {
      this.detailLoading.set(false);
    }
  }

  protected closeDetail(): void {
    this.detailDialogOpen = false;
    this.selectedDetail.set(null);
  }

  protected formatDate(value: string): string {
    const parts = parseISODateParts(value);
    if (!parts) {
      return '-';
    }

    return `${pad2(parts.day)}/${pad2(parts.month)}/${parts.year}`;
  }

  protected formatCompletedAt(value: string): string {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return '-';
    }

    return this.dateTimeFormatter.format(date);
  }

  protected formatCompletedTime(value: string): string {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return '-';
    }

    return this.timeFormatter.format(date);
  }

  protected blockCountLabel(item: HistoryListItem): string {
    return item.blockCount === 1 ? '1 bloco' : `${item.blockCount} blocos`;
  }

  protected participantCountLabel(count: number | null | undefined): string {
    if (count === null || count === undefined) {
      return '';
    }

    return count === 1 ? '1 aluno' : `${count} alunos`;
  }

  protected hasParticipation(detail: HistoryDetail): boolean {
    return detail.participantCount !== null || detail.participantNames.length > 0;
  }

  protected hasNotes(detail: HistoryDetail): boolean {
    return detail.notes !== null && detail.notes.trim() !== '';
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
      if (error.status === 400) {
        this.errorMessage.set('Período inválido.');
        return;
      }
      if (error.status === 404) {
        this.errorMessage.set('Registro de histórico não encontrado.');
        return;
      }
    }

    this.errorMessage.set('Não foi possível carregar o histórico. Tente novamente.');
  }
}

function groupHistory(items: HistoryListItem[], formatter: Intl.DateTimeFormat): HistoryGroup[] {
  const grouped = new Map<string, HistoryListItem[]>();
  const sorted = items.slice().sort(compareHistoryItems);

  for (const item of sorted) {
    grouped.set(item.trainingDate, [...(grouped.get(item.trainingDate) ?? []), item]);
  }

  return Array.from(grouped.entries()).map(([date, groupItems]) => ({
    date,
    label: formatGroupLabel(date, formatter),
    items: groupItems,
  }));
}

function compareHistoryItems(first: HistoryListItem, second: HistoryListItem): number {
  if (first.trainingDate !== second.trainingDate) {
    return second.trainingDate.localeCompare(first.trainingDate);
  }

  return second.completedAt.localeCompare(first.completedAt);
}

function formatGroupLabel(value: string, formatter: Intl.DateTimeFormat): string {
  const parts = parseISODateParts(value);
  if (!parts) {
    return value;
  }

  return capitalize(formatter.format(new Date(parts.year, parts.month - 1, parts.day)));
}

function monthRange(date: Date): { from: string; to: string } {
  const first = startOfMonth(date);
  const last = new Date(first.getFullYear(), first.getMonth() + 1, 0);

  return {
    from: dateToISODate(first),
    to: dateToISODate(last),
  };
}

function startOfMonth(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), 1);
}

function dateToISODate(date: Date): string {
  return `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())}`;
}

function parseISODateParts(value: string): { year: number; month: number; day: number } | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) {
    return null;
  }

  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const date = new Date(year, month - 1, day);
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
    return null;
  }

  return { year, month, day };
}

function pad2(value: number): string {
  return value.toString().padStart(2, '0');
}

function capitalize(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
