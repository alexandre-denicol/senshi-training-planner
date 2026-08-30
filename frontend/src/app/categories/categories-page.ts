import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { Category, CategoryApiService } from './category-api.service';

type ConfirmationKind = 'status' | 'delete';

interface ConfirmationState {
  kind: ConfirmationKind;
  category: Category;
  nextActive?: boolean;
}

@Component({
  imports: [ButtonModule, CommonModule, DialogModule, FormsModule, InputTextModule],
  selector: 'app-categories-page',
  styleUrl: './categories-page.css',
  templateUrl: './categories-page.html',
})
export class CategoriesPage implements OnInit {
  private readonly api = inject(CategoryApiService);
  private readonly router = inject(Router);
  private readonly dateFormatter = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short' });

  protected readonly maxNameLength = 120;

  protected readonly categories = signal<Category[]>([]);
  protected readonly loading = signal(true);
  protected readonly submitting = signal(false);
  protected readonly rowActionId = signal<string | null>(null);
  protected readonly errorMessage = signal('');
  protected readonly successMessage = signal('');
  protected readonly confirmation = signal<ConfirmationState | null>(null);
  protected readonly hasCategories = computed(() => this.categories().length > 0);

  protected createDialogOpen = false;
  protected editDialogOpen = false;
  protected selectedCategory: Category | null = null;
  protected createForm = this.emptyForm();
  protected editForm = this.emptyForm();

  async ngOnInit(): Promise<void> {
    await this.loadCategories();
  }

  protected async loadCategories(): Promise<void> {
    this.loading.set(true);
    this.errorMessage.set('');

    try {
      this.categories.set(await this.api.list());
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

  protected openEditDialog(category: Category): void {
    this.clearMessages();
    this.selectedCategory = category;
    this.editForm = { name: category.name };
    this.editDialogOpen = true;
  }

  protected closeEditDialog(): void {
    this.editDialogOpen = false;
    this.selectedCategory = null;
    this.editForm = this.emptyForm();
  }

  protected formValid(form: { name: string }): boolean {
    return form.name.trim() !== '';
  }

  protected confirmStatus(category: Category): void {
    this.clearMessages();
    this.confirmation.set({
      kind: 'status',
      category,
      nextActive: !category.active,
    });
  }

  protected confirmDelete(category: Category): void {
    this.clearMessages();
    this.confirmation.set({ kind: 'delete', category });
  }

  protected cancelConfirmation(): void {
    this.confirmation.set(null);
  }

  protected async createCategory(): Promise<void> {
    if (!this.formValid(this.createForm) || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.clearMessages();

    try {
      const category = await this.api.create({ name: this.createForm.name });
      this.categories.update((items) => [...items, category].sort(compareCategories));
      this.successMessage.set('Categoria cadastrada com sucesso.');
      this.closeCreateDialog();
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.submitting.set(false);
    }
  }

  protected async updateCategory(): Promise<void> {
    if (!this.selectedCategory || !this.formValid(this.editForm) || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.clearMessages();

    try {
      const updated = await this.api.update(this.selectedCategory.id, { name: this.editForm.name });
      this.replaceCategory(updated);
      this.successMessage.set('Categoria atualizada com sucesso.');
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
    this.rowActionId.set(confirmation.category.id);
    this.clearMessages();

    try {
      if (confirmation.kind === 'delete') {
        await this.api.delete(confirmation.category.id);
        this.categories.update((items) => items.filter((item) => item.id !== confirmation.category.id));
        this.successMessage.set('Categoria excluída com sucesso.');
      } else {
        const updated = await this.api.setStatus(confirmation.category.id, confirmation.nextActive === true);
        this.replaceCategory(updated);
        this.successMessage.set(updated.active ? 'Categoria ativada com sucesso.' : 'Categoria desativada com sucesso.');
      }
      this.confirmation.set(null);
    } catch (error) {
      if (confirmation.kind === 'delete' && this.isConflict(error)) {
        this.errorMessage.set('Esta categoria está sendo utilizada por um ou mais blocos e não pode ser excluída.');
        return;
      }
      await this.handleRequestError(error);
    } finally {
      this.rowActionId.set(null);
      this.submitting.set(false);
    }
  }

  protected formatDate(value: string): string {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return '-';
    }

    return this.dateFormatter.format(date);
  }

  protected statusLabel(category: Category): string {
    return category.active ? 'Ativa' : 'Inativa';
  }

  protected confirmationTitle(): string {
    const confirmation = this.confirmation();
    if (!confirmation) {
      return '';
    }
    if (confirmation.kind === 'delete') {
      return `Excluir ${confirmation.category.name}?`;
    }

    return confirmation.nextActive ? `Ativar ${confirmation.category.name}?` : `Desativar ${confirmation.category.name}?`;
  }

  protected confirmationText(): string {
    const confirmation = this.confirmation();
    if (!confirmation) {
      return '';
    }
    if (confirmation.kind === 'delete') {
      return 'A categoria será permanentemente removida.';
    }
    if (confirmation.nextActive) {
      return 'A categoria ficará disponível para uso futuro.';
    }

    return 'A categoria ficará indisponível para uso futuro, mas não será excluída.';
  }

  private replaceCategory(updated: Category): void {
    this.categories.update((items) =>
      items.map((item) => item.id === updated.id ? updated : item).sort(compareCategories),
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
        this.errorMessage.set('Categoria não encontrada.');
        return;
      }
      if (error.status === 409) {
        this.errorMessage.set('Já existe uma categoria com este nome.');
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

  private emptyForm() {
    return { name: '' };
  }
}

function compareCategories(first: Category, second: Category): number {
  return first.name.localeCompare(second.name, 'pt-BR', { sensitivity: 'base' });
}
