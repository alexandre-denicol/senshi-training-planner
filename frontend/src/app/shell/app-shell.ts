import { CommonModule } from '@angular/common';
import { Component, ElementRef, HostListener, ViewChild, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { AuthService } from '../auth/auth.service';
import { AuthUser } from '../auth/auth.models';

interface NavItem {
  label: string;
  icon: string;
  path?: string;
  adminOnly?: boolean;
}

const navItems: NavItem[] = [
  { label: 'Dashboard', icon: 'pi pi-home', path: '/app/dashboard' },
  { label: 'Agenda', icon: 'pi pi-calendar', path: '/app/agenda' },
  { label: 'Treinos', icon: 'pi pi-list-check', path: '/app/treinos' },
  { label: 'Blocos', icon: 'pi pi-th-large', path: '/app/blocos' },
  { label: 'Categorias', icon: 'pi pi-tags', path: '/app/categorias' },
  { label: 'Histórico', icon: 'pi pi-clock', path: '/app/historico' },
  { label: 'Alunos', icon: 'pi pi-users', path: '/app/alunos' },
  { label: 'Professores', icon: 'pi pi-users', path: '/app/professores', adminOnly: true },
];

@Component({
  imports: [ButtonModule, CommonModule, FormsModule, RouterLink, RouterLinkActive, RouterOutlet],
  selector: 'app-shell',
  styleUrl: './app-shell.css',
  templateUrl: './app-shell.html',
})
export class AppShell {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  @ViewChild('menuButton') private readonly menuButton?: ElementRef<HTMLButtonElement>;

  protected readonly drawerOpen = signal(false);
  protected readonly passwordDialogOpen = signal(false);
  protected readonly passwordLoading = signal(false);
  protected readonly passwordError = signal('');
  protected passwordForm = { currentPassword: '', newPassword: '', confirmPassword: '' };
  protected readonly user = computed(() => this.auth.currentUser() as AuthUser);
  protected readonly visibleNavItems = computed(() => {
    const user = this.auth.currentUser();
    return navItems.filter((item) => !item.adminOnly || user?.role === 'ADMIN');
  });

  @HostListener('document:keydown.escape')
  protected onEscape(): void {
    if (this.drawerOpen()) {
      this.closeDrawer();
    }
  }

  protected closeDrawer(restoreFocus = true): void {
    const wasOpen = this.drawerOpen();
    this.drawerOpen.set(false);
    if (wasOpen && restoreFocus) {
      this.menuButton?.nativeElement.focus();
    }
  }

  protected closeDrawerAfterNavigation(): void {
    this.closeDrawer(false);
  }

  protected async logout(): Promise<void> {
    await this.auth.logout();
    await this.router.navigateByUrl('/login');
  }

  protected openPasswordDialog(): void { this.passwordError.set(''); this.passwordDialogOpen.set(true); }
  protected closePasswordDialog(): void { this.passwordDialogOpen.set(false); this.clearPasswordForm(); }
  protected passwordFormValid(): boolean {
    return this.passwordForm.currentPassword.length > 0 && this.passwordForm.newPassword.length >= 8 && this.passwordForm.newPassword === this.passwordForm.confirmPassword;
  }
  protected async changePassword(): Promise<void> {
    if (this.passwordLoading() || !this.passwordFormValid()) return;
    this.passwordLoading.set(true); this.passwordError.set('');
    try {
      await this.auth.changePassword(this.passwordForm.currentPassword, this.passwordForm.newPassword);
      this.clearPasswordForm(); this.passwordDialogOpen.set(false);
      await this.router.navigateByUrl('/login', { state: { message: 'Senha alterada. Entre novamente com a nova senha.' } });
    } catch { this.passwordError.set('Não foi possível alterar a senha. Verifique a senha atual e tente novamente.'); }
    finally { this.passwordLoading.set(false); }
  }
  private clearPasswordForm(): void { this.passwordForm = { currentPassword: '', newPassword: '', confirmPassword: '' }; }
}
