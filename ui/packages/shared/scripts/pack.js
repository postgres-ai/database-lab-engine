const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const TMP_DIR = 'build-tmp';
const DIST_DIR = 'dist';
const PACKAGE_JSON = 'package.json';

function cleanTmp() {
  if (fs.existsSync(TMP_DIR)) {
    fs.rmSync(TMP_DIR, { recursive: true, force: true });
  }
}

function run(cmd, options = {}) {
  console.log(`$ ${cmd}`);
  execSync(cmd, { stdio: 'inherit', ...options });
}

function copyDistToTmp() {
  run(`rsync -a ${DIST_DIR}/ ${TMP_DIR}/`);
}

function copyExtraFiles() {
  const extras = ['react-app-env.d.ts'];
  extras.forEach((file) => {
    if (fs.existsSync(file)) {
      fs.copyFileSync(file, path.join(TMP_DIR, file));
    }
  });
}

function sanitizePackageJson() {
  const original = JSON.parse(fs.readFileSync(PACKAGE_JSON, 'utf8'));
  const cleaned = {
    name: original.name,
    version: original.version,
    description: original.description,
    author: original.author,
    license: original.license,
    main: original.main || 'index.js',
    types: original.types || 'index.d.ts',
    peerDependencies: original.peerDependencies,
    dependencies: original.dependencies,
  };

  fs.writeFileSync(
    path.join(TMP_DIR, 'package.json'),
    JSON.stringify(cleaned, null, 2),
    'utf8'
  );
}

function pack() {
  run('npm pack', { cwd: TMP_DIR });
  const tarball = fs.readdirSync(TMP_DIR).find(f => f.endsWith('.tgz'));
  fs.renameSync(path.join(TMP_DIR, tarball), path.join('.', tarball));
  console.log(`✅ Packed to ./${tarball}`);
}

function buildTmpAndPack() {
  cleanTmp();
  run('pnpm run build');
  copyDistToTmp();
  sanitizePackageJson();
  pack();
  cleanTmp();
}

buildTmpAndPack();