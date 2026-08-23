import { CommonModule } from '@angular/common';
import { Component, computed, inject, signal } from '@angular/core';
import { Router, RouterLink, RouterOutlet } from '@angular/router';
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
  { label: 'Agenda', icon: 'pi pi-calendar', path: '/app/agenda' },
  { label: 'Treinos', icon: 'pi pi-list-check', path: '/app/treinos' },
  { label: 'Blocos', icon: 'pi pi-th-large', path: '/app/blocos' },
  { label: 'Categorias', icon: 'pi pi-tags', path: '/app/categorias' },
  { label: 'Histórico', icon: 'pi pi-clock', path: '/app/historico' },
  { label: 'Professores', icon: 'pi pi-users', path: '/app/professores', adminOnly: true },
];

@Component({
  imports: [ButtonModule, CommonModule, RouterLink, RouterOutlet],
  selector: 'app-shell',
  styleUrl: './app-shell.css',
  templateUrl: './app-shell.html',
})
export class AppShell {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  protected readonly drawerOpen = signal(false);
  protected readonly user = computed(() => this.auth.currentUser() as AuthUser);
  protected readonly visibleNavItems = computed(() => {
    const user = this.auth.currentUser();
    return navItems.filter((item) => !item.adminOnly || user?.role === 'ADMIN');
  });

  protected closeDrawer(): void {
    this.drawerOpen.set(false);
  }

  protected async logout(): Promise<void> {
    await this.auth.logout();
    await this.router.navigateByUrl('/login');
  }
}
