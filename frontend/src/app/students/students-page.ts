import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { Student, StudentApiService } from './student-api.service';

type ConfirmationKind = 'status';

interface ConfirmationState {
  kind: ConfirmationKind;
  student: Student;
  nextActive?: boolean;
}

interface StudentForm {
  name: string;
  birthDate?: string;
  phone?: string;
  guardianName?: string;
  guardianPhone?: string;
  notes?: string;
}

@Component({
  imports: [
    ButtonModule,
    CommonModule,
    DialogModule,
    FormsModule,
    InputTextModule,
  ],
  selector: 'app-students-page',
  styleUrl: './students-page.css',
  templateUrl: './students-page.html',
})
export class StudentsPage implements OnInit {
  private readonly api = inject(StudentApiService);
  private readonly router = inject(Router);
  private readonly dateFormatter = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short' });

  protected readonly maxNameLength = 120;
  protected readonly maxPhoneLength = 30;
  protected readonly maxNotesLength = 2000;

  protected readonly students = signal<Student[]>([]);
  protected readonly loading = signal(true);
  protected readonly submitting = signal(false);
  protected readonly rowActionId = signal<string | null>(null);
  protected readonly errorMessage = signal('');
  protected readonly successMessage = signal('');
  protected readonly confirmation = signal<ConfirmationState | null>(null);

  protected createDialogOpen = false;
  protected editDialogOpen = false;
  protected selectedStudent: Student | null = null;

  protected createForm = this.emptyForm();
  protected editForm = this.emptyForm();

  protected readonly hasStudents = computed(() => this.students().length > 0);

  async ngOnInit(): Promise<void> {
    await this.loadStudents();
  }

  protected async loadStudents(): Promise<void> {
    this.loading.set(true);
    this.errorMessage.set('');

    try {
      this.students.set(await this.api.list());
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

  protected openEditDialog(student: Student): void {
    this.clearMessages();
    this.selectedStudent = student;
    this.editForm = {
      name: student.name,
      birthDate: student.birthDate,
      phone: student.phone,
      guardianName: student.guardianName,
      guardianPhone: student.guardianPhone,
      notes: student.notes,
    };
    this.editDialogOpen = true;
  }

  protected closeEditDialog(): void {
    this.editDialogOpen = false;
    this.selectedStudent = null;
    this.editForm = this.emptyForm();
  }

  protected formValid(form: StudentForm): boolean {
    return form.name.trim() !== '';
  }

  protected confirmStatus(student: Student): void {
    this.clearMessages();
    this.confirmation.set({
      kind: 'status',
      student,
      nextActive: !student.active,
    });
  }

  protected cancelConfirmation(): void {
    this.confirmation.set(null);
  }

  protected async createStudent(): Promise<void> {
    if (!this.formValid(this.createForm) || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.clearMessages();

    try {
      const student = await this.api.create(this.createForm);
      this.students.update((items) => [...items, student].sort(compareStudents));
      this.successMessage.set('Aluno cadastrado com sucesso.');
      this.closeCreateDialog();
    } catch (error) {
      await this.handleRequestError(error);
    } finally {
      this.submitting.set(false);
    }
  }

  protected async updateStudent(): Promise<void> {
    if (!this.selectedStudent || !this.formValid(this.editForm) || this.submitting()) {
      return;
    }

    this.submitting.set(true);
    this.clearMessages();

    try {
      const updated = await this.api.update(this.selectedStudent.id, this.editForm);
      this.replaceStudent(updated);
      this.successMessage.set('Aluno atualizado com sucesso.');
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
    this.rowActionId.set(confirmation.student.id);
    this.clearMessages();

    try {
      const updated = await this.api.setStatus(confirmation.student.id, confirmation.nextActive === true);
      this.replaceStudent(updated);
      this.successMessage.set(updated.active ? 'Aluno ativado com sucesso.' : 'Aluno desativado com sucesso.');
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

  protected statusLabel(student: Student): string {
    return student.active ? 'Ativo' : 'Inativo';
  }

  protected confirmationTitle(): string {
    const confirmation = this.confirmation();
    if (!confirmation) {
      return '';
    }

    return confirmation.nextActive ? `Ativar ${confirmation.student.name}?` : `Desativar ${confirmation.student.name}?`;
  }

  protected confirmationText(): string {
    const confirmation = this.confirmation();
    if (!confirmation) {
      return '';
    }
    if (confirmation.nextActive) {
      return 'O aluno poderá ser selecionado para treinos.';
    }

    return 'O aluno não poderá ser selecionado para treinos.';
  }

  private replaceStudent(updated: Student): void {
    this.students.update((items) =>
      items.map((item) => item.id === updated.id ? updated : item).sort(compareStudents),
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
        this.errorMessage.set('Aluno não encontrado.');
        return;
      }
    }

    this.errorMessage.set('Não foi possível concluir a operação. Tente novamente.');
  }

  private clearMessages(): void {
    this.errorMessage.set('');
    this.successMessage.set('');
  }

  private emptyForm(): StudentForm {
    return { name: '' };
  }
}

function compareStudents(first: Student, second: Student): number {
  return first.name.localeCompare(second.name, 'pt-BR', { sensitivity: 'base' });
}
