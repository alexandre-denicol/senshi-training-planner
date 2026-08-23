import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { Block, BlockApiService } from '../blocks/block-api.service';
import { WorkoutBlock, WorkoutDetail, WorkoutListItem, WorkoutApiService } from './workout-api.service';

type ConfirmationKind = 'status' | 'delete';

interface ConfirmationState {
  kind: ConfirmationKind;
  workout: WorkoutListItem;
  nextActive?: boolean;
}

interface WorkoutForm {
  name: string;
  selectedBlockId: string;
  blocks: WorkoutBlock[];
}

@Component({
  imports: [ButtonModule, CommonModule, DialogModule, FormsModule, InputTextModule],
  selector: 'app-workouts-page',
  styleUrl: './workouts-page.css',
  templateUrl: './workouts-page.html',
})
export class WorkoutsPage implements OnInit {
  private readonly api = inject(WorkoutApiService);
  private readonly blockApi = inject(BlockApiService);
  private readonly router = inject(Router);
  private readonly dateFormatter = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short' });

  protected readonly workouts = signal<WorkoutListItem[]>([]);
  protected readonly blocks = signal<Block[]>([]);
  protected readonly loading = signal(true);
  protected readonly submitting = signal(false);
  protected readonly detailLoading = signal(false);
  protected readonly rowActionId = signal<string | null>(null);
  protected readonly errorMessage = signal('');
  protected readonly successMessage = signal('');
  protected readonly confirmation = signal<ConfirmationState | null>(null);
  protected readonly hasWorkouts = computed(() => this.workouts().length > 0);
  protected readonly activeBlocks = computed(() => this.blocks().filter((block) => block.active));
  protected readonly hasActiveBlocks = computed(() => this.activeBlocks().length > 0);

  protected createDialogOpen = false;
  protected editDialogOpen = false;
  protected selectedWorkout: WorkoutListItem | null = null;
  protected createForm = this.emptyForm();
  protected editForm = this.emptyForm();

  async ngOnInit(): Promise<void> {
    await this.loadData();
  }

  protected async loadData(): Promise<void> {
    this.loading.set(true);
    this.errorMessage.set('');

    try {
      const [workouts, blocks] = await Promise.all([
        this.api.list(),
        this.blockApi.list(),
      ]);
      this.workouts.set(workouts);
      this.blocks.set(blocks);
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.loading.set(false);
    }
  }

  protected openCreateDialog(): void {
    this.clearMessages();
    if (!this.hasActiveBlocks()) {
      this.errorMessage.set('Cadastre ou ative pelo menos um bloco antes de criar um treino.');
      return;
    }

    this.createForm = this.emptyForm();
    this.createDialogOpen = true;
  }

  protected closeCreateDialog(): void {
    this.createDialogOpen = false;
    this.createForm = this.emptyForm();
  }

  protected async openEditDialog(workout: WorkoutListItem): Promise<void> {
    this.clearMessages();
    this.selectedWorkout = workout;
    this.detailLoading.set(true);
    this.editDialogOpen = true;

    try {
      const detail = await this.api.getById(workout.id);
      this.editForm = this.formFromDetail(detail);
    } catch (error) {
      this.editDialogOpen = false;
      this.selectedWorkout = null;
      await this.handleRequestError(error);
    } finally {
      this.detailLoading.set(false);
    }
  }

  protected closeEditDialog(): void {
    this.editDialogOpen = false;
    this.selectedWorkout = null;
    this.editForm = this.emptyForm();
  }

  protected formValid(form: WorkoutForm): boolean {
    return form.name.trim() !== '' && form.blocks.length > 0;
  }

  protected availableBlocks(form: WorkoutForm): Block[] {
    const selected = new Set(form.blocks.map((block) => block.id));
    return this.activeBlocks().filter((block) => !selected.has(block.id));
  }

  protected addSelectedBlock(form: WorkoutForm): void {
    if (!form.selectedBlockId) {
      return;
    }

    const block = this.activeBlocks().find((item) => item.id === form.selectedBlockId);
    if (!block || form.blocks.some((item) => item.id === block.id)) {
      form.selectedBlockId = '';
      return;
    }

    form.blocks = [
      ...form.blocks,
      this.workoutBlockFromBlock(block, form.blocks.length + 1),
    ];
    form.selectedBlockId = '';
    this.reposition(form);
  }

  protected removeBlock(form: WorkoutForm, index: number): void {
    form.blocks = form.blocks.filter((_, itemIndex) => itemIndex !== index);
    this.reposition(form);
  }

  protected moveBlockUp(form: WorkoutForm, index: number): void {
    if (index <= 0) {
      return;
    }
    [form.blocks[index - 1], form.blocks[index]] = [form.blocks[index], form.blocks[index - 1]];
    this.reposition(form);
  }

  protected moveBlockDown(form: WorkoutForm, index: number): void {
    if (index >= form.blocks.length - 1) {
      return;
    }
    [form.blocks[index], form.blocks[index + 1]] = [form.blocks[index + 1], form.blocks[index]];
    this.reposition(form);
  }

  protected confirmStatus(workout: WorkoutListItem): void {
    this.clearMessages();
    this.confirmation.set({
      kind: 'status',
      workout,
      nextActive: !workout.active,
    });
  }

  protected confirmDelete(workout: WorkoutListItem): void {
    this.clearMessages();
    this.confirmation.set({ kind: 'delete', workout });
  }

  protected cancelConfirmation(): void {
    this.confirmation.set(null);
  }

  protected async createWorkout(): Promise<void> {
    if (!this.formValid(this.createForm) || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.clearMessages();

    try {
      const workout = await this.api.create(this.requestFromForm(this.createForm));
      this.workouts.update((items) => [...items, this.listItemFromDetail(workout)].sort(compareWorkouts));
      this.successMessage.set('Treino cadastrado com sucesso.');
      this.closeCreateDialog();
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.submitting.set(false);
    }
  }

  protected async updateWorkout(): Promise<void> {
    if (!this.selectedWorkout || !this.formValid(this.editForm) || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.clearMessages();

    try {
      const updated = await this.api.update(this.selectedWorkout.id, this.requestFromForm(this.editForm));
      this.replaceWorkout(this.listItemFromDetail(updated));
      this.successMessage.set('Treino atualizado com sucesso.');
      this.closeEditDialog();
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.submitting.set(false);
    }
  }

  protected async applyConfirmation(): Promise<void> {
    const confirmation = this.confirmation();
    if (!confirmation || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.rowActionId.set(confirmation.workout.id);
    this.clearMessages();

    try {
      if (confirmation.kind === 'delete') {
        await this.api.delete(confirmation.workout.id);
        this.workouts.update((items) => items.filter((item) => item.id !== confirmation.workout.id));
        this.successMessage.set('Treino excluído com sucesso.');
      } else {
        const updated = await this.api.setStatus(confirmation.workout.id, confirmation.nextActive === true);
        this.replaceWorkout(updated);
        this.successMessage.set(updated.active ? 'Treino ativado com sucesso.' : 'Treino desativado com sucesso.');
      }
      this.confirmation.set(null);
    } catch (error) {
      if (confirmation.kind === 'delete' && this.isConflict(error)) {
        this.errorMessage.set('Este treino está sendo utilizado na agenda e não pode ser excluído.');
        return;
      }
      await this.handleRequestError(error);
    } finally {
      this.rowActionId.set(null);
      this.submitting.set(false);
    }
  }

  protected blockStatusLabel(block: WorkoutBlock): string {
    return block.active ? '' : 'Inativo';
  }

  protected formatBlockCount(workout: WorkoutListItem): string {
    return workout.blockCount === 1 ? '1 bloco' : `${workout.blockCount} blocos`;
  }

  protected formatDate(value: string): string {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return '-';
    }

    return this.dateFormatter.format(date);
  }

  protected statusLabel(workout: WorkoutListItem): string {
    return workout.active ? 'Ativo' : 'Inativo';
  }

  protected confirmationTitle(): string {
    const confirmation = this.confirmation();
    if (!confirmation) {
      return '';
    }
    if (confirmation.kind === 'delete') {
      return `Excluir ${confirmation.workout.name}?`;
    }

    return confirmation.nextActive ? `Ativar ${confirmation.workout.name}?` : `Desativar ${confirmation.workout.name}?`;
  }

  protected confirmationText(): string {
    const confirmation = this.confirmation();
    if (!confirmation) {
      return '';
    }
    if (confirmation.kind === 'delete') {
      return 'O treino será permanentemente removido.';
    }
    if (confirmation.nextActive) {
      return 'O treino ficará disponível para uso futuro.';
    }

    return 'O treino ficará indisponível para uso futuro.';
  }

  private requestFromForm(form: WorkoutForm) {
    return {
      name: form.name,
      blockIds: form.blocks.map((block) => block.id),
    };
  }

  private replaceWorkout(updated: WorkoutListItem): void {
    this.workouts.update((items) =>
      items.map((item) => item.id === updated.id ? updated : item).sort(compareWorkouts),
    );
  }

  private listItemFromDetail(detail: WorkoutDetail): WorkoutListItem {
    return {
      id: detail.id,
      name: detail.name,
      active: detail.active,
      blockCount: detail.blocks.length,
      createdAt: detail.createdAt,
      updatedAt: detail.updatedAt,
    };
  }

  private formFromDetail(detail: WorkoutDetail): WorkoutForm {
    return {
      name: detail.name,
      selectedBlockId: '',
      blocks: detail.blocks
        .slice()
        .sort((first, second) => first.position - second.position)
        .map((block, index) => ({ ...block, position: index + 1 })),
    };
  }

  private workoutBlockFromBlock(block: Block, position: number): WorkoutBlock {
    return {
      id: block.id,
      name: block.name,
      active: block.active,
      position,
      category: block.category,
    };
  }

  private reposition(form: WorkoutForm): void {
    form.blocks = form.blocks.map((block, index) => ({ ...block, position: index + 1 }));
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
        this.errorMessage.set('Treino não encontrado.');
        return;
      }
      if (error.status === 409) {
        this.errorMessage.set('Já existe um treino com este nome.');
        return;
      }
    }

    this.errorMessage.set('Não foi possível concluir a operação. Tente novamente.');
  }

  private isConflict(error: unknown): boolean {
    return error instanceof HttpErrorResponse && error.status === 409;
  }

  private clearMessages(): void {
    this.errorMessage.set('');
    this.successMessage.set('');
  }

  private emptyForm(): WorkoutForm {
    return { name: '', selectedBlockId: '', blocks: [] };
  }
}

function compareWorkouts(first: WorkoutListItem, second: WorkoutListItem): number {
  return first.name.localeCompare(second.name, 'pt-BR', { sensitivity: 'base' });
}
