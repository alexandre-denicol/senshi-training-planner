import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { AuthService } from '../auth/auth.service';
import { WorkoutApiService, WorkoutDetail, WorkoutListItem } from '../workouts/workout-api.service';
import { CompletionDetails, ScheduleApiService, ScheduleEntry, ScheduleWorkout } from './schedule-api.service';

interface ScheduleForm {
  scheduledDate: string;
  workoutId: string;
}

interface ScheduleGroup {
  date: string;
  label: string;
  entries: ScheduleEntry[];
}

interface ConfirmationState {
  entry: ScheduleEntry;
}

interface CompletionForm {
  participantCount: string | number | null;
  participantName: string;
  participantNames: string[];
  notes: string;
}

@Component({
  imports: [ButtonModule, CommonModule, DialogModule, FormsModule, InputTextModule],
  selector: 'app-schedule-page',
  styleUrl: './schedule-page.css',
  templateUrl: './schedule-page.html',
})
export class SchedulePage implements OnInit {
  private readonly api = inject(ScheduleApiService);
  private readonly workoutApi = inject(WorkoutApiService);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly monthFormatter = new Intl.DateTimeFormat('pt-BR', { month: 'long', year: 'numeric' });
  private readonly groupFormatter = new Intl.DateTimeFormat('pt-BR', { weekday: 'long', day: '2-digit', month: 'long' });

  protected readonly entries = signal<ScheduleEntry[]>([]);
  protected readonly workouts = signal<WorkoutListItem[]>([]);
  protected readonly currentMonth = signal(startOfMonth(new Date()));
  protected readonly loading = signal(true);
  protected readonly submitting = signal(false);
  protected readonly rowActionId = signal<string | null>(null);
  protected readonly errorMessage = signal('');
  protected readonly successMessage = signal('');
  protected readonly confirmation = signal<ConfirmationState | null>(null);
  protected readonly completionFormError = signal('');
  protected readonly workoutDetail = signal<WorkoutDetail | null>(null);
  protected readonly workoutDetailLoading = signal(false);
  protected readonly workoutDetailError = signal('');
  protected readonly workoutDetailLoadingEntryId = signal<string | null>(null);
  protected readonly canManageSchedule = computed(() => this.auth.currentUser() !== null);
  protected readonly hasEntries = computed(() => this.entries().length > 0);
  protected readonly activeWorkouts = computed(() => this.workouts().filter((workout) => workout.active));
  protected readonly hasActiveWorkouts = computed(() => this.activeWorkouts().length > 0);
  protected readonly periodLabel = computed(() => capitalize(this.monthFormatter.format(this.currentMonth())));
  protected readonly groups = computed(() => groupEntries(this.entries(), this.groupFormatter));

  protected createDialogOpen = false;
  protected editDialogOpen = false;
  protected readonly completeDialogOpen = signal(false);
  protected readonly workoutDetailDialogOpen = signal(false);
  protected selectedEntry: ScheduleEntry | null = null;
  protected completionEntry: ScheduleEntry | null = null;
  protected workoutDetailEntry: ScheduleEntry | null = null;
  protected createForm = this.emptyForm();
  protected editForm = this.emptyForm();
  protected completionForm = this.emptyCompletionForm();

  async ngOnInit(): Promise<void> {
    await this.loadData();
  }

  protected async loadData(): Promise<void> {
    this.loading.set(true);
    this.errorMessage.set('');

    try {
      const range = monthRange(this.currentMonth());
      const entriesPromise = this.api.list(range.from, range.to);
      const [entries, workouts] = await Promise.all([entriesPromise, this.workoutApi.list()]);
      this.entries.set(entries);
      this.workouts.set(workouts);
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.loading.set(false);
    }
  }

  protected async previousMonth(): Promise<void> {
    this.currentMonth.update((date) => startOfMonth(new Date(date.getFullYear(), date.getMonth() - 1, 1)));
    await this.loadData();
  }

  protected async nextMonth(): Promise<void> {
    this.currentMonth.update((date) => startOfMonth(new Date(date.getFullYear(), date.getMonth() + 1, 1)));
    await this.loadData();
  }

  protected async goToToday(): Promise<void> {
    this.currentMonth.set(startOfMonth(new Date()));
    await this.loadData();
  }

  protected openCreateDialog(): void {
    this.clearMessages();
    if (!this.hasActiveWorkouts()) {
      this.errorMessage.set('Cadastre ou ative pelo menos um treino antes de agendar.');
      return;
    }

    this.createForm = { scheduledDate: todayISODate(), workoutId: '' };
    this.createDialogOpen = true;
  }

  protected closeCreateDialog(): void {
    this.createDialogOpen = false;
    this.createForm = this.emptyForm();
  }

  protected openEditDialog(entry: ScheduleEntry): void {
    this.clearMessages();
    this.selectedEntry = entry;
    this.editForm = { scheduledDate: entry.scheduledDate, workoutId: entry.workout.id };
    this.editDialogOpen = true;
  }

  protected closeEditDialog(): void {
    this.editDialogOpen = false;
    this.selectedEntry = null;
    this.editForm = this.emptyForm();
  }

  protected formValid(form: ScheduleForm): boolean {
    return form.scheduledDate !== '' && form.workoutId !== '';
  }

  protected availableWorkouts(currentWorkout: ScheduleWorkout | null): ScheduleWorkout[] {
    const options = this.activeWorkouts().map((workout) => ({
      id: workout.id,
      name: workout.name,
      active: workout.active,
    }));

    if (currentWorkout && !currentWorkout.active && !options.some((workout) => workout.id === currentWorkout.id)) {
      options.push(currentWorkout);
    }

    return options.sort(compareWorkoutOptions);
  }

  protected async createEntry(): Promise<void> {
    if (!this.formValid(this.createForm) || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.clearMessages();

    try {
      await this.api.create({ workoutId: this.createForm.workoutId, scheduledDate: this.createForm.scheduledDate });
      this.successMessage.set('Treino agendado com sucesso.');
      this.closeCreateDialog();
      await this.loadData();
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.submitting.set(false);
    }
  }

  protected async updateEntry(): Promise<void> {
    if (!this.selectedEntry || !this.formValid(this.editForm) || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.clearMessages();

    try {
      await this.api.update(this.selectedEntry.id, { workoutId: this.editForm.workoutId, scheduledDate: this.editForm.scheduledDate });
      this.successMessage.set('Agendamento atualizado com sucesso.');
      this.closeEditDialog();
      await this.loadData();
    } catch (error) {
      if (this.isConflict(error)) {
        this.errorMessage.set('Treinos já realizados não podem ser alterados ou removidos.');
        return;
      }
      await this.handleRequestError(error);
    } finally {
      this.submitting.set(false);
    }
  }

  protected confirmComplete(entry: ScheduleEntry): void {
    this.clearMessages();
    this.completionEntry = entry;
    this.completionForm = this.emptyCompletionForm();
    this.completeDialogOpen.set(true);
  }

  protected closeCompleteDialog(): void {
    this.completeDialogOpen.set(false);
    this.completionEntry = null;
    this.completionForm = this.emptyCompletionForm();
    this.completionFormError.set('');
  }

  protected async openWorkoutDetail(entry: ScheduleEntry): Promise<void> {
    if (this.isCompleted(entry) || this.workoutDetailLoadingEntryId() === entry.id) {
      return;
    }

    this.clearMessages();
    this.workoutDetailEntry = entry;
    this.workoutDetail.set(null);
    this.workoutDetailError.set('');
    this.workoutDetailDialogOpen.set(true);
    this.workoutDetailLoading.set(true);
    this.workoutDetailLoadingEntryId.set(entry.id);

    try {
      this.workoutDetail.set(await this.workoutApi.getById(entry.workout.id));
    } catch (error) {
      await this.handleWorkoutDetailError(error);
    } finally {
      this.workoutDetailLoading.set(false);
      this.workoutDetailLoadingEntryId.set(null);
    }
  }

  protected closeWorkoutDetailDialog(): void {
    this.workoutDetailDialogOpen.set(false);
    this.workoutDetailEntry = null;
    this.workoutDetail.set(null);
    this.workoutDetailError.set('');
    this.workoutDetailLoading.set(false);
    this.workoutDetailLoadingEntryId.set(null);
  }

  protected addParticipantName(input?: HTMLInputElement): void {
    const name = (input?.value ?? this.completionForm.participantName).trim();
    this.completionFormError.set('');

    if (name === '') {
      this.completionFormError.set('Informe o nome antes de adicionar.');
      return;
    }
    if (Array.from(name).length > 120) {
      this.completionFormError.set('Use no máximo 120 caracteres por nome.');
      return;
    }
    if (this.completionForm.participantNames.length >= 100) {
      this.completionFormError.set('Use no máximo 100 participantes registrados.');
      return;
    }

    this.completionForm = {
      ...this.completionForm,
      participantName: '',
      participantNames: [...this.completionForm.participantNames, name],
    };
    if (input) {
      input.value = '';
    }
  }

  protected addParticipantNameFromKeyboard(event: Event, input: HTMLInputElement): void {
    event.preventDefault();
    this.addParticipantName(input);
  }

  protected removeParticipantName(index: number): void {
    this.completionForm = {
      ...this.completionForm,
      participantNames: this.completionForm.participantNames.filter((_, currentIndex) => currentIndex !== index),
    };
  }

  protected completionFormValid(): boolean {
    const count = this.completionCountText();
    if (count === '') {
      return this.completionNotesValid();
    }

    const value = Number(count);
    return Number.isInteger(value) && value >= 0 && value <= 500 && this.completionNotesValid();
  }

  protected completionNotesCounter(): string {
    return `${Array.from(this.completionForm.notes).length} / 2000`;
  }

  protected async completeEntry(): Promise<void> {
    if (!this.completionEntry || !this.completionFormValid() || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.rowActionId.set(this.completionEntry.id);
    this.clearMessages();
    this.completionFormError.set('');

    try {
      await this.api.complete(this.completionEntry.id, this.completionPayload());
      this.successMessage.set('Treino registrado como realizado.');
      this.closeCompleteDialog();
      await this.loadData();
    } catch (error) {
      if (this.isConflict(error)) {
        this.errorMessage.set('Este treino já foi registrado como realizado.');
        return;
      }
      if (error instanceof HttpErrorResponse && error.status === 400) {
        this.errorMessage.set('Verifique os dados informados.');
        return;
      }
      this.errorMessage.set('Não foi possível registrar o treino como realizado. Tente novamente.');
    } finally {
      this.rowActionId.set(null);
      this.submitting.set(false);
    }
  }

  protected confirmDelete(entry: ScheduleEntry): void {
    this.clearMessages();
    this.confirmation.set({ entry });
  }

  protected cancelConfirmation(): void {
    this.confirmation.set(null);
  }

  protected async applyConfirmation(): Promise<void> {
    const confirmation = this.confirmation();
    if (!confirmation || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.rowActionId.set(confirmation.entry.id);
    this.clearMessages();

    try {
      await this.api.delete(confirmation.entry.id);
      this.entries.update((items) => items.filter((item) => item.id !== confirmation.entry.id));
      this.successMessage.set('Agendamento removido com sucesso.');
      this.confirmation.set(null);
    } catch (error) {
      if (this.isConflict(error)) {
        this.errorMessage.set('Treinos já realizados não podem ser alterados ou removidos.');
        return;
      }
      await this.handleRequestError(error);
    } finally {
      this.rowActionId.set(null);
      this.submitting.set(false);
    }
  }

  protected formatDate(value: string): string {
    const parts = parseISODateParts(value);
    if (!parts) {
      return '-';
    }

    return `${pad2(parts.day)}/${pad2(parts.month)}/${parts.year}`;
  }

  protected workoutOptionLabel(workout: ScheduleWorkout): string {
    return workout.active ? workout.name : `${workout.name} (inativo)`;
  }

  protected isCompleted(entry: ScheduleEntry): boolean {
    return Boolean(entry.completedAt);
  }

  protected confirmationTitle(): string {
    const confirmation = this.confirmation();
    if (!confirmation) {
      return '';
    }

    return `Remover ${confirmation.entry.workout.name} da agenda de ${this.formatDate(confirmation.entry.scheduledDate)}?`;
  }

  protected confirmationText(): string {
    const confirmation = this.confirmation();
    if (!confirmation) {
      return '';
    }

    return 'Isso remove apenas o agendamento. O treino permanece no catálogo.';
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
        this.errorMessage.set('Verifique os dados informados.');
        return;
      }
      if (error.status === 404) {
        this.errorMessage.set('Agendamento não encontrado.');
        return;
      }
      if (error.status === 409) {
        this.errorMessage.set('Este treino já está agendado para esta data.');
        return;
      }
    }

    this.errorMessage.set('Não foi possível concluir a operação. Tente novamente.');
  }

  private async handleWorkoutDetailError(error: unknown): Promise<void> {
    if (error instanceof HttpErrorResponse) {
      if (error.status === 401) {
        await this.router.navigateByUrl('/login');
        return;
      }
      if (error.status === 403) {
        await this.router.navigateByUrl('/app');
        return;
      }
      if (error.status === 404) {
        this.workoutDetailError.set('Treino não encontrado.');
        return;
      }
    }

    this.workoutDetailError.set('Não foi possível carregar os detalhes do treino. Tente novamente.');
  }

  private isConflict(error: unknown): boolean {
    return error instanceof HttpErrorResponse && error.status === 409;
  }

  private clearMessages(): void {
    this.errorMessage.set('');
    this.successMessage.set('');
  }

  private emptyForm(): ScheduleForm {
    return { scheduledDate: '', workoutId: '' };
  }

  private emptyCompletionForm(): CompletionForm {
    return { participantCount: '', participantName: '', participantNames: [], notes: '' };
  }

  private completionPayload(): CompletionDetails {
    const payload: CompletionDetails = {};
    const count = this.completionCountText();
    if (count !== '') {
      payload.participantCount = Number(count);
    }
    if (this.completionForm.participantNames.length > 0) {
      payload.participantNames = this.completionForm.participantNames;
    }
    const notes = this.completionForm.notes.trim();
    if (notes !== '') {
      payload.notes = notes;
    }

    return payload;
  }

  private completionCountText(): string {
    const count = this.completionForm.participantCount;
    if (count === null || count === undefined) {
      return '';
    }

    return String(count).trim();
  }

  private completionNotesValid(): boolean {
    return Array.from(this.completionForm.notes).length <= 2000;
  }
}

function groupEntries(entries: ScheduleEntry[], formatter: Intl.DateTimeFormat): ScheduleGroup[] {
  const grouped = new Map<string, ScheduleEntry[]>();
  const sorted = entries.slice().sort(compareEntries);

  for (const entry of sorted) {
    grouped.set(entry.scheduledDate, [...(grouped.get(entry.scheduledDate) ?? []), entry]);
  }

  return Array.from(grouped.entries()).map(([date, items]) => ({
    date,
    label: formatGroupLabel(date, formatter),
    entries: items,
  }));
}

function compareEntries(first: ScheduleEntry, second: ScheduleEntry): number {
  if (first.scheduledDate !== second.scheduledDate) {
    return first.scheduledDate.localeCompare(second.scheduledDate);
  }

  return first.workout.name.localeCompare(second.workout.name, 'pt-BR', { sensitivity: 'base' });
}

function compareWorkoutOptions(first: ScheduleWorkout, second: ScheduleWorkout): number {
  return first.name.localeCompare(second.name, 'pt-BR', { sensitivity: 'base' });
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

function todayISODate(): string {
  return dateToISODate(new Date());
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
