import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { Block, BlockApiService } from '../blocks/block-api.service';
import { Category, CategoryApiService } from '../categories/category-api.service';
import { ScheduleApiService } from '../schedule/schedule-api.service';
import { WorkoutBlock, WorkoutDetail, WorkoutListItem, WorkoutApiService } from './workout-api.service';

type ConfirmationKind = 'status' | 'delete';
type AddBlockMode = 'select' | 'create';

interface ConfirmationState {
  kind: ConfirmationKind;
  workout: WorkoutListItem;
  nextActive?: boolean;
}

interface WorkoutForm {
  name: string;
  blocks: WorkoutBlock[];
}

interface BlockForm {
  name: string;
  categoryId: string;
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
  private readonly categoryApi = inject(CategoryApiService);
  private readonly scheduleApi = inject(ScheduleApiService);
  private readonly router = inject(Router);
  private readonly dateFormatter = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short' });

  protected readonly workouts = signal<WorkoutListItem[]>([]);
  protected readonly blocks = signal<Block[]>([]);
  protected readonly categories = signal<Category[]>([]);
  protected readonly loading = signal(true);
  protected readonly submitting = signal(false);
  protected readonly detailLoading = signal(false);
  protected readonly rowActionId = signal<string | null>(null);
  protected readonly errorMessage = signal('');
  protected readonly successMessage = signal('');
  protected readonly builderErrorMessage = signal('');
  protected readonly addBlockErrorMessage = signal('');
  protected readonly addBlockFeedback = signal('');
  protected readonly categoryErrorMessage = signal('');
  protected readonly scheduleErrorMessage = signal('');
  protected readonly confirmation = signal<ConfirmationState | null>(null);
  protected readonly hasWorkouts = computed(() => this.workouts().length > 0);
  protected readonly activeBlocks = computed(() => this.blocks().filter((block) => block.active));
  protected readonly hasActiveBlocks = computed(() => this.activeBlocks().length > 0);
  protected readonly activeCategories = computed(() => this.categories().filter((category) => category.active));
  protected readonly hasActiveCategories = computed(() => this.activeCategories().length > 0);

  protected createDialogOpen = false;
  protected editDialogOpen = false;
  protected addBlockDialogOpen = false;
  protected categoryDialogOpen = false;
  protected postSaveDialogOpen = false;
  protected addBlockMode: AddBlockMode = 'select';
  protected selectedWorkout: WorkoutListItem | null = null;
  protected activeBuilderForm: WorkoutForm | null = null;
  protected createdWorkout: WorkoutDetail | null = null;
  protected blockSearch = '';
  protected createForm = this.emptyForm();
  protected editForm = this.emptyForm();
  protected blockCreateForm = this.emptyBlockForm();
  protected categoryCreateForm = { name: '' };
  protected scheduleForm = { scheduledDate: todayISODate() };

  async ngOnInit(): Promise<void> {
    await this.loadData();
  }

  protected async loadData(): Promise<void> {
    this.loading.set(true);
    this.errorMessage.set('');

    try {
      const [workouts, blocks, categories] = await Promise.all([
        this.api.list(),
        this.blockApi.list(),
        this.categoryApi.list(),
      ]);
      this.workouts.set(workouts);
      this.blocks.set(blocks);
      this.categories.set(categories);
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.loading.set(false);
    }
  }

  protected openCreateDialog(): void {
    this.clearMessages();
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

  protected availableBlocks(form: WorkoutForm | null): Block[] {
    if (!form) {
      return [];
    }

    const selected = new Set(form.blocks.map((block) => block.id));
    return this.activeBlocks().filter((block) => !selected.has(block.id));
  }

  protected filteredAvailableBlocks(): Block[] {
    const query = this.blockSearch.trim().toLocaleLowerCase('pt-BR');
    const blocks = this.availableBlocks(this.activeBuilderForm);
    if (query === '') {
      return blocks;
    }

    return blocks.filter((block) =>
      block.name.toLocaleLowerCase('pt-BR').includes(query)
      || block.category.name.toLocaleLowerCase('pt-BR').includes(query),
    );
  }

  protected openAddBlockDialog(form: WorkoutForm): void {
    this.clearContextMessages();
    this.activeBuilderForm = form;
    this.blockSearch = '';
    this.blockCreateForm = this.emptyBlockForm();
    this.addBlockMode = 'select';
    this.addBlockDialogOpen = true;
  }

  protected closeAddBlockDialog(): void {
    this.addBlockDialogOpen = false;
    this.activeBuilderForm = null;
    this.blockSearch = '';
    this.blockCreateForm = this.emptyBlockForm();
    this.clearContextMessages();
  }

  protected showCreateBlock(): void {
    this.clearContextMessages();
    this.blockCreateForm = this.emptyBlockForm();
    this.addBlockMode = 'create';
  }

  protected showBlockSelection(): void {
    this.clearContextMessages();
    this.addBlockMode = 'select';
  }

  protected addExistingBlock(block: Block): void {
    const form = this.activeBuilderForm;
    if (!form || form.blocks.some((item) => item.id === block.id) || !block.active) {
      return;
    }

    form.blocks = [
      ...form.blocks,
      this.workoutBlockFromBlock(block, form.blocks.length + 1),
    ];
    this.reposition(form);
    this.addBlockFeedback.set('Bloco adicionado ao treino.');
  }

  protected blockFormValid(): boolean {
    return this.blockCreateForm.name.trim() !== '' && this.blockCreateForm.categoryId !== '';
  }

  protected categoryFormValid(): boolean {
    return this.categoryCreateForm.name.trim() !== '';
  }

  protected openCategoryDialog(): void {
    this.categoryErrorMessage.set('');
    this.categoryCreateForm = { name: '' };
    this.categoryDialogOpen = true;
  }

  protected closeCategoryDialog(): void {
    this.categoryDialogOpen = false;
    this.categoryCreateForm = { name: '' };
    this.categoryErrorMessage.set('');
  }

  protected async createCategoryInBuilder(): Promise<void> {
    if (!this.categoryFormValid() || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.categoryErrorMessage.set('');

    try {
      const category = await this.categoryApi.create({ name: this.categoryCreateForm.name });
      this.categories.update((items) => [...items, category].sort(compareCategories));
      this.blockCreateForm.categoryId = category.id;
      this.addBlockFeedback.set('Categoria criada.');
      this.closeCategoryDialog();
    } catch (error) {
      this.handleCategoryRequestError(error);
    } finally {
      this.submitting.set(false);
    }
  }

  protected async createBlockInBuilder(): Promise<void> {
    const form = this.activeBuilderForm;
    if (!form || !this.blockFormValid() || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.addBlockErrorMessage.set('');
    this.addBlockFeedback.set('');

    try {
      const block = await this.blockApi.create({
        name: this.blockCreateForm.name,
        categoryId: this.blockCreateForm.categoryId,
      });
      this.blocks.update((items) => [...items, block].sort(compareBlocks));
      this.addExistingBlock(block);
      this.successMessage.set('Bloco criado e adicionado ao treino.');
      this.closeAddBlockDialog();
    } catch (error) {
      this.handleBlockRequestError(error);
    } finally {
      this.submitting.set(false);
    }
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
      this.createdWorkout = workout;
      this.scheduleForm = { scheduledDate: todayISODate() };
      this.closeCreateDialog();
      this.postSaveDialogOpen = true;
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

  protected closePostSaveDialog(): void {
    this.postSaveDialogOpen = false;
    this.createdWorkout = null;
    this.scheduleForm = { scheduledDate: todayISODate() };
    this.scheduleErrorMessage.set('');
  }

  protected async scheduleCreatedWorkout(): Promise<void> {
    if (!this.createdWorkout || this.scheduleForm.scheduledDate === '' || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.scheduleErrorMessage.set('');

    try {
      await this.scheduleApi.create({
        workoutId: this.createdWorkout.id,
        scheduledDate: this.scheduleForm.scheduledDate,
      });
      this.successMessage.set('Treino agendado com sucesso.');
      this.closePostSaveDialog();
    } catch (error) {
      if (error instanceof HttpErrorResponse && error.status === 409) {
        this.scheduleErrorMessage.set('Este treino já está agendado para esta data.');
      } else if (error instanceof HttpErrorResponse && error.status === 400) {
        this.scheduleErrorMessage.set('Verifique os dados informados.');
      } else {
        this.scheduleErrorMessage.set('Não foi possível concluir a operação. Tente novamente.');
      }
    } finally {
      this.submitting.set(false);
    }
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

  private handleBlockRequestError(error: unknown): void {
    if (error instanceof HttpErrorResponse) {
      if (error.status === 400) {
        this.addBlockErrorMessage.set('Verifique os dados informados.');
        return;
      }
      if (error.status === 409) {
        this.addBlockErrorMessage.set('Já existe um bloco com este nome nesta categoria.');
        return;
      }
    }

    this.addBlockErrorMessage.set('Não foi possível concluir a operação. Tente novamente.');
  }

  private handleCategoryRequestError(error: unknown): void {
    if (error instanceof HttpErrorResponse) {
      if (error.status === 400) {
        this.categoryErrorMessage.set('Verifique os dados informados.');
        return;
      }
      if (error.status === 409) {
        this.categoryErrorMessage.set('Já existe uma categoria com este nome.');
        return;
      }
    }

    this.categoryErrorMessage.set('Não foi possível concluir a operação. Tente novamente.');
  }

  private isConflict(error: unknown): boolean {
    return error instanceof HttpErrorResponse && error.status === 409;
  }

  private clearMessages(): void {
    this.errorMessage.set('');
    this.successMessage.set('');
    this.builderErrorMessage.set('');
    this.clearContextMessages();
    this.scheduleErrorMessage.set('');
  }

  private clearContextMessages(): void {
    this.addBlockErrorMessage.set('');
    this.addBlockFeedback.set('');
    this.categoryErrorMessage.set('');
  }

  private emptyForm(): WorkoutForm {
    return { name: '', blocks: [] };
  }

  private emptyBlockForm(): BlockForm {
    return { name: '', categoryId: '' };
  }
}

function compareWorkouts(first: WorkoutListItem, second: WorkoutListItem): number {
  return first.name.localeCompare(second.name, 'pt-BR', { sensitivity: 'base' });
}

function compareBlocks(first: Block, second: Block): number {
  const categoryComparison = first.category.name.localeCompare(second.category.name, 'pt-BR', { sensitivity: 'base' });
  if (categoryComparison !== 0) {
    return categoryComparison;
  }

  return first.name.localeCompare(second.name, 'pt-BR', { sensitivity: 'base' });
}

function compareCategories(first: Category, second: Category): number {
  return first.name.localeCompare(second.name, 'pt-BR', { sensitivity: 'base' });
}

function todayISODate(): string {
  const date = new Date();
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}
