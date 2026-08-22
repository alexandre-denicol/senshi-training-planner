import { CommonModule } from '@angular/common';
import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { InputTextModule } from 'primeng/inputtext';
import { AuthService } from '../auth/auth.service';

@Component({
  imports: [ButtonModule, CommonModule, FormsModule, InputTextModule],
  selector: 'app-login-page',
  styleUrl: './login.page.css',
  templateUrl: './login.page.html',
})
export class LoginPage {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  protected email = '';
  protected password = '';
  protected loading = signal(false);
  protected errorMessage = signal('');

  protected async submit(): Promise<void> {
    if (this.loading()) {
      return;
    }

    this.loading.set(true);
    this.errorMessage.set('');

    try {
      await this.auth.login(this.email, this.password);
      await this.router.navigateByUrl('/app');
    } catch {
      this.errorMessage.set('E-mail ou senha inválidos.');
    } finally {
      this.loading.set(false);
    }
  }
}
