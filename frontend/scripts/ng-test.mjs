import { spawnSync } from 'node:child_process';

const forwardedArgs = process.argv.slice(2).filter((arg) => arg !== '--run');
const executable = process.platform === 'win32' ? 'npx.cmd' : 'npx';
const result = spawnSync(
  executable,
  ['ng', 'test', '--watch=false', ...forwardedArgs],
  { stdio: 'inherit' },
);

process.exit(result.status ?? 1);
