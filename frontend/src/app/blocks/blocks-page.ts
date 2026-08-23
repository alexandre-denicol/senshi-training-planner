import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { Category, CategoryApiService } from '../categories/category-api.service';
import { Block, BlockApiService } from './block-api.service';

type ConfirmationKind = 'status' | 'delete';

interface ConfirmationState {
  kind: ConfirmationKind;
  block: Block;
  nextActive?: boolean;
}

interface BlockForm {
  name: string;
  categoryId: string;
}

@Component({
  imports: [ButtonModule, CommonModule, DialogModule, FormsModule, InputTextModule],
  selector: 'app-blocks-page',
  styleUrl: './blocks-page.css',
  templateUrl: './blocks-page.html',
})
export class BlocksPage implements OnInit {
  private readonly api = inject(BlockApiService);
  private readonly categoryApi = inject(CategoryApiService);
  private readonly router = inject(Router);
  private readonly dateFormatter = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short' });

  protected readonly blocks = signal<Block[]>([]);
  protected readonly categories = signal<Category[]>([]);
  protected readonly loading = signal(true);
  protected readonly submitting = signal(false);
  protected readonly rowActionId = signal<string | null>(null);
  protected readonly errorMessage = signal('');
  protected readonly successMessage = signal('');
  protected readonly confirmation = signal<ConfirmationState | null>(null);
  protected readonly hasBlocks = computed(() => this.blocks().length > 0);
  protected readonly activeCategories = computed(() => this.categories().filter((category) => category.active));
  protected readonly hasActiveCategories = computed(() => this.activeCategories().length > 0);

  protected createDialogOpen = false;
  protected editDialogOpen = false;
  protected selectedBlock: Block | null = null;
  protected createForm = this.emptyForm();
  protected editForm = this.emptyForm();

  async ngOnInit(): Promise<void> {
    await this.loadData();
  }

  protected async loadData(): Promise<void> {
    this.loading.set(true);
    this.errorMessage.set('');

    try {
      const [blocks, categories] = await Promise.all([
        this.api.list(),
        this.categoryApi.list(),
      ]);
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
    if (!this.hasActiveCategories()) {
      this.errorMessage.set('Cadastre ou ative uma categoria antes de criar blocos.');
      return;
    }

    this.createForm = this.emptyForm();
    this.createDialogOpen = true;
  }

  protected closeCreateDialog(): void {
    this.createDialogOpen = false;
    this.createForm = this.emptyForm();
  }

  protected openEditDialog(block: Block): void {
    this.clearMessages();
    this.selectedBlock = block;
    this.editForm = {
      name: block.name,
      categoryId: this.categoryIsActive(block.category.id) ? block.category.id : '',
    };
    this.editDialogOpen = true;
  }

  protected closeEditDialog(): void {
    this.editDialogOpen = false;
    this.selectedBlock = null;
    this.editForm = this.emptyForm();
  }

  protected formValid(form: BlockForm): boolean {
    return form.name.trim() !== '' && form.categoryId !== '';
  }

  protected confirmStatus(block: Block): void {
    this.clearMessages();
    this.confirmation.set({
      kind: 'status',
      block,
      nextActive: !block.active,
    });
  }

  protected confirmDelete(block: Block): void {
    this.clearMessages();
    this.confirmation.set({ kind: 'delete', block });
  }

  protected cancelConfirmation(): void {
    this.confirmation.set(null);
  }

  protected async createBlock(): Promise<void> {
    if (!this.formValid(this.createForm) || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.clearMessages();

    try {
      const block = await this.api.create({ name: this.createForm.name, categoryId: this.createForm.categoryId });
      this.blocks.update((items) => [...items, block].sort(compareBlocks));
      this.successMessage.set('Bloco cadastrado com sucesso.');
      this.closeCreateDialog();
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.submitting.set(false);
    }
  }

  protected async updateBlock(): Promise<void> {
    if (!this.selectedBlock || !this.formValid(this.editForm) || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.clearMessages();

    try {
      const updated = await this.api.update(this.selectedBlock.id, {
        name: this.editForm.name,
        categoryId: this.editForm.categoryId,
      });
      this.replaceBlock(updated);
      this.successMessage.set('Bloco atualizado com sucesso.');
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
    this.rowActionId.set(confirmation.block.id);
    this.clearMessages();

    try {
      if (confirmation.kind === 'delete') {
        await this.api.delete(confirmation.block.id);
        this.blocks.update((items) => items.filter((item) => item.id !== confirmation.block.id));
        this.successMessage.set('Bloco excluído com sucesso.');
      } else {
        const updated = await this.api.setStatus(confirmation.block.id, confirmation.nextActive === true);
        this.replaceBlock(updated);
        this.successMessage.set(updated.active ? 'Bloco ativado com sucesso.' : 'Bloco desativado com sucesso.');
      }
      this.confirmation.set(null);
    } catch (error) {
      if (confirmation.kind === 'delete' && this.isConflict(error)) {
        this.errorMessage.set('Este bloco está sendo utilizado por um ou mais treinos e não pode ser excluído.');
        return;
      }
      await this.handleRequestError(error);
    } finally {
      this.rowActionId.set(null);
      this.submitting.set(false);
    }
  }

  protected categoryIsActive(categoryId: string): boolean {
    return this.activeCategories().some((category) => category.id === categoryId);
  }

  protected formatDate(value: string): string {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return '-';
    }

    return this.dateFormatter.format(date);
  }

  protected statusLabel(block: Block): string {
    return block.active ? 'Ativo' : 'Inativo';
  }

  protected confirmationTitle(): string {
    const confirmation = this.confirmation();
    if (!confirmation) {
      return '';
    }
    if (confirmation.kind === 'delete') {
      return `Excluir ${confirmation.block.name}?`;
    }

    return confirmation.nextActive ? `Ativar ${confirmation.block.name}?` : `Desativar ${confirmation.block.name}?`;
  }

  protected confirmationText(): string {
    const confirmation = this.confirmation();
    if (!confirmation) {
      return '';
    }
    if (confirmation.kind === 'delete') {
      return 'O bloco será permanentemente removido.';
    }
    if (confirmation.nextActive) {
      return 'O bloco ficará disponível para uso futuro.';
    }

    return 'O bloco ficará indisponível para uso futuro, mas não será excluído.';
  }

  private replaceBlock(updated: Block): void {
    this.blocks.update((items) =>
      items.map((item) => item.id === updated.id ? updated : item).sort(compareBlocks),
    );
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
        this.errorMessage.set('Bloco não encontrado.');
        return;
      }
      if (error.status === 409) {
        this.errorMessage.set('Já existe um bloco com este nome nesta categoria.');
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

  private emptyForm(): BlockForm {
    return { name: '', categoryId: '' };
  }
}

function compareBlocks(first: Block, second: Block): number {
  const categoryCompare = first.category.name.localeCompare(second.category.name, 'pt-BR', { sensitivity: 'base' });
  if (categoryCompare !== 0) {
    return categoryCompare;
  }

  return first.name.localeCompare(second.name, 'pt-BR', { sensitivity: 'base' });
}
