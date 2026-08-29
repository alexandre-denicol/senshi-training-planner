import { Page, test } from '@playwright/test';

export interface E2ECredentials {
  email: string;
  password: string;
}

export function adminCredentials(): E2ECredentials | null {
  return credentials('E2E_ADMIN_EMAIL', 'E2E_ADMIN_PASSWORD');
}

export function professorCredentials(): E2ECredentials | null {
  return credentials('E2E_PROFESSOR_EMAIL', 'E2E_PROFESSOR_PASSWORD');
}

export function mutationAllowed(): boolean {
  return process.env.E2E_ALLOW_MUTATION === 'true';
}

export async function requireBackend(page: Page): Promise<void> {
  if (!(await backendAvailable(page))) {
    test.skip(true, 'Backend is not reachable through the Angular proxy.');
  }
}

export function requireCredentials(credentials: E2ECredentials | null, label: string): asserts credentials is E2ECredentials {
  if (!credentials) {
    test.skip(true, `${label} E2E credentials are not configured.`);
  }
}

export function requireMutationSafety(): void {
  if (!mutationAllowed()) {
    test.skip(true, 'Mutable E2E tests require E2E_ALLOW_MUTATION=true and an isolated test database.');
  }
  if (!hasSafeE2EDatabaseMarker()) {
    test.skip(true, 'Mutable E2E tests require E2E_DATABASE_URL pointing at the local senshi_e2e PostgreSQL database.');
  }
  if (!hasSafeE2EBackendTarget()) {
    test.skip(true, 'Mutable E2E tests require E2E_API_PROXY_TARGET pointing at the isolated E2E backend.');
  }
}

export function uniqueName(prefix: string): string {
  const random = Math.random().toString(36).slice(2, 8);
  return `E2E ${prefix} ${Date.now()} ${random}`;
}

export function isoDateInCurrentMonth(day = 15): string {
  const now = new Date();
  const date = new Date(now.getFullYear(), now.getMonth(), day);
  return `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())}`;
}

export function brDateFromISO(value: string): string {
  const [year, month, day] = value.split('-');
  return `${day}/${month}/${year}`;
}

async function backendAvailable(page: Page): Promise<boolean> {
  try {
    const response = await page.request.get('/api/auth/me', { failOnStatusCode: false, timeout: 5_000 });
    return response.status() === 200 || response.status() === 401;
  } catch {
    return false;
  }
}

function credentials(emailKey: string, passwordKey: string): E2ECredentials | null {
  const email = process.env[emailKey];
  const password = process.env[passwordKey];
  if (!email || !password) {
    return null;
  }

  return { email, password };
}

function pad2(value: number): string {
  return value.toString().padStart(2, '0');
}

function hasSafeE2EDatabaseMarker(): boolean {
  const databaseURL = process.env.E2E_DATABASE_URL;
  if (!databaseURL) {
    return false;
  }

  try {
    const parsed = new URL(databaseURL);
    const hostname = parsed.hostname.toLowerCase();
    return (
      (hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '[::1]') &&
      parsed.pathname === '/senshi_e2e' &&
      parsed.searchParams.get('sslmode') === 'disable'
    );
  } catch {
    return false;
  }
}

function hasSafeE2EBackendTarget(): boolean {
  const target = process.env.E2E_API_PROXY_TARGET || process.env.API_PROXY_TARGET;
  if (!target) {
    return false;
  }

  try {
    const parsed = new URL(target);
    const hostname = parsed.hostname.toLowerCase();
    return (
      (hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '[::1]') &&
      parsed.port !== '' &&
      parsed.port !== '8080' &&
      parsed.port !== '18080' &&
      parsed.port !== '4200'
    );
  } catch {
    return false;
  }
}
