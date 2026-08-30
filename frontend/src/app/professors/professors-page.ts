import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { Professor, ProfessorApiService } from './professor-api.service';

type ConfirmationKind = 'status' | 'delete';

interface ConfirmationState {
  kind: ConfirmationKind;
  professor: Professor;
  nextActive?: boolean;
}

@Component({
  imports: [
    ButtonModule,
    CommonModule,
    DialogModule,
    FormsModule,
    InputTextModule,
  ],
  selector: 'app-professors-page',
  styleUrl: './professors-page.css',
  templateUrl: './professors-page.html',
})
export class ProfessorsPage implements OnInit {
  private readonly api = inject(ProfessorApiService);
  private readonly router = inject(Router);
  private readonly dateFormatter = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short' });

  protected readonly maxNameLength = 120;

  protected readonly professors = signal<Professor[]>([]);
  protected readonly loading = signal(true);
  protected readonly submitting = signal(false);
  protected readonly rowActionId = signal<string | null>(null);
  protected readonly errorMessage = signal('');
  protected readonly successMessage = signal('');
  protected readonly confirmation = signal<ConfirmationState | null>(null);

  protected createDialogOpen = false;
  protected editDialogOpen = false;
  protected passwordDialogOpen = false;
  protected selectedProfessor: Professor | null = null;

  protected createForm = this.emptyCreateForm();
  protected editForm = this.emptyEditForm();
  protected passwordForm = this.emptyPasswordForm();

  protected readonly hasProfessors = computed(() => this.professors().length > 0);

  async ngOnInit(): Promise<void> {
    await this.loadProfessors();
  }

  protected async loadProfessors(): Promise<void> {
    this.loading.set(true);
    this.errorMessage.set('');

    try {
      this.professors.set(await this.api.list());
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.loading.set(false);
    }
  }

  protected openCreateDialog(): void {
    this.clearMessages();
    this.createForm = this.emptyCreateForm();
    this.createDialogOpen = true;
  }

  protected closeCreateDialog(): void {
    this.createDialogOpen = false;
    this.createForm = this.emptyCreateForm();
  }

  protected openEditDialog(professor: Professor): void {
    this.clearMessages();
    this.selectedProfessor = professor;
    this.editForm = { name: professor.name, email: professor.email };
    this.editDialogOpen = true;
  }

  protected closeEditDialog(): void {
    this.editDialogOpen = false;
    this.selectedProfessor = null;
    this.editForm = this.emptyEditForm();
  }

  protected openPasswordDialog(professor: Professor): void {
    this.clearMessages();
    this.selectedProfessor = professor;
    this.passwordForm = this.emptyPasswordForm();
    this.passwordDialogOpen = true;
  }

  protected closePasswordDialog(): void {
    this.passwordDialogOpen = false;
    this.selectedProfessor = null;
    this.passwordForm = this.emptyPasswordForm();
  }

  protected confirmStatus(professor: Professor): void {
    this.clearMessages();
    this.confirmation.set({
      kind: 'status',
      professor,
      nextActive: !professor.active,
    });
  }

  protected confirmDelete(professor: Professor): void {
    this.clearMessages();
    this.confirmation.set({ kind: 'delete', professor });
  }

  protected cancelConfirmation(): void {
    this.confirmation.set(null);
  }

  protected createFormValid(): boolean {
    return this.createForm.name.trim() !== '' &&
      this.createForm.email.trim() !== '' &&
      this.createForm.password !== '' &&
      this.createForm.confirmPassword !== '' &&
      this.createForm.password === this.createForm.confirmPassword;
  }

  protected editFormValid(): boolean {
    return this.editForm.name.trim() !== '' && this.editForm.email.trim() !== '';
  }

  protected passwordFormValid(): boolean {
    return this.passwordForm.password !== '' &&
      this.passwordForm.confirmPassword !== '' &&
      this.passwordForm.password === this.passwordForm.confirmPassword;
  }

  protected async createProfessor(): Promise<void> {
    if (!this.createFormValid() || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.clearMessages();

    try {
      const professor = await this.api.create({
        name: this.createForm.name,
        email: this.createForm.email,
        password: this.createForm.password,
      });
      this.professors.update((items) => [...items, professor].sort(compareProfessors));
      this.successMessage.set('Professor cadastrado com sucesso.');
      this.closeCreateDialog();
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.submitting.set(false);
    }
  }

  protected async updateProfessor(): Promise<void> {
    if (!this.selectedProfessor || !this.editFormValid() || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.clearMessages();

    try {
      const updated = await this.api.update(this.selectedProfessor.id, {
        name: this.editForm.name,
        email: this.editForm.email,
      });
      this.replaceProfessor(updated);
      this.successMessage.set('Professor atualizado com sucesso.');
      this.closeEditDialog();
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.submitting.set(false);
    }
  }

  protected async changePassword(): Promise<void> {
    if (!this.selectedProfessor || !this.passwordFormValid() || this.submitting()) {
      return;
    }

    const professorName = this.selectedProfessor.name;
    const professorID = this.selectedProfessor.id;
    this.submitting.set(true);
    this.clearMessages();

    try {
      await this.api.setPassword(professorID, this.passwordForm.password);
      this.successMessage.set(`Senha de ${professorName} alterada com sucesso.`);
      this.closePasswordDialog();
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.passwordForm = this.emptyPasswordForm();
      this.submitting.set(false);
    }
  }

  protected async applyConfirmation(): Promise<void> {
    const confirmation = this.confirmation();
    if (!confirmation || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.rowActionId.set(confirmation.professor.id);
    this.clearMessages();

    try {
      if (confirmation.kind === 'delete') {
        await this.api.delete(confirmation.professor.id);
        this.professors.update((items) => items.filter((item) => item.id !== confirmation.professor.id));
        this.successMessage.set('Professor excluído com sucesso.');
      } else {
        const updated = await this.api.setStatus(confirmation.professor.id, confirmation.nextActive === true);
        this.replaceProfessor(updated);
        this.successMessage.set(updated.active ? 'Professor ativado com sucesso.' : 'Professor desativado com sucesso.');
      }
      this.confirmation.set(null);
    } catch (error) {
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

  protected statusLabel(professor: Professor): string {
    return professor.active ? 'Ativo' : 'Inativo';
  }

  protected confirmationTitle(): string {
    const confirmation = this.confirmation();
    if (!confirmation) {
      return '';
    }
    if (confirmation.kind === 'delete') {
      return `Excluir ${confirmation.professor.name}?`;
    }

    return confirmation.nextActive ? `Ativar ${confirmation.professor.name}?` : `Desativar ${confirmation.professor.name}?`;
  }

  protected confirmationText(): string {
    const confirmation = this.confirmation();
    if (!confirmation) {
      return '';
    }
    if (confirmation.kind === 'delete') {
      return 'A conta será permanentemente removida.';
    }
    if (confirmation.nextActive) {
      return 'O professor poderá acessar o sistema novamente.';
    }

    return 'As sessões ativas desse professor serão encerradas.';
  }

  private replaceProfessor(updated: Professor): void {
    this.professors.update((items) =>
      items.map((item) => item.id === updated.id ? updated : item).sort(compareProfessors),
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
        this.errorMessage.set('Professor não encontrado.');
        return;
      }
      if (error.status === 409) {
        this.errorMessage.set('Já existe uma conta com este e-mail.');
        return;
      }
    }

    this.errorMessage.set('Não foi possível concluir a operação. Tente novamente.');
  }

  private clearMessages(): void {
    this.errorMessage.set('');
    this.successMessage.set('');
  }

  private emptyCreateForm() {
    return {
      name: '',
      email: '',
      password: '',
      confirmPassword: '',
    };
  }

  private emptyEditForm() {
    return {
      name: '',
      email: '',
    };
  }

  private emptyPasswordForm() {
    return {
      password: '',
      confirmPassword: '',
    };
  }
}

function compareProfessors(first: Professor, second: Professor): number {
  return first.name.localeCompare(second.name, 'pt-BR', { sensitivity: 'base' }) ||
    first.email.localeCompare(second.email, 'pt-BR', { sensitivity: 'base' });
}
