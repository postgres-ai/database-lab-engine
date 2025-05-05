const fs = require('fs');
const path = require('path');
const glob = require('glob');

const OUT_DIR = 'dist';

const PATTERNS = [
  '**/*.scss',
  '**/*.module.scss',
  '**/*.json',
  'react-app-env.d.ts',
];

const files = PATTERNS.flatMap(pattern =>
  glob.sync(pattern, {
    cwd: '.',
    ignore: ['node_modules/**', 'dist/**'],
    nodir: true,
  })
);

files.forEach((file) => {
  const from = path.resolve(file);
  const to = path.join(OUT_DIR, file);
  const dir = path.dirname(to);
  fs.mkdirSync(dir, { recursive: true });
  fs.copyFileSync(from, to);
});

console.log(`✅ Copied ${files.length} assets to dist`);