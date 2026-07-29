'use strict'

const assert = require('node:assert/strict')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const { spawnSync } = require('node:child_process')
const test = require('node:test')

const helperPath = path.join(__dirname, 'install-config.js')
const unixOnly = { skip: process.platform === 'win32' }

function fixture(t, homeName = 'home') {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'tocreate-config-test-'))
  const home = path.join(root, homeName)
  fs.mkdirSync(home, { recursive: true, mode: 0o700 })
  t.after(() => fs.rmSync(root, { recursive: true, force: true }))
  return { root, home }
}

function writeFile(filePath, content, mode = 0o600) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true, mode: 0o700 })
  fs.writeFileSync(filePath, content, { mode })
}

function writeResponse(root, client, values = {}) {
  const responsePath = path.join(root, `${client}-response.json`)
  writeFile(
    responsePath,
    `${JSON.stringify({
      data: {
        client,
        provider_name: 'ToCreate',
        endpoint: 'https://api.example.com/v1',
        api_key: 'sk-test-secret',
        model: '',
        ...values,
      },
    })}\n`,
  )
  return responsePath
}

function helperEnvironment(home, shell, extra = {}) {
  const env = { ...process.env }
  for (const key of [
    'CLAUDE_CONFIG_DIR',
    'CODEX_HOME',
    'GEMINI_CLI_HOME',
    'TOCREATE_API_KEY',
  ]) {
    delete env[key]
  }
  return {
    ...env,
    HOME: home,
    USERPROFILE: home,
    SHELL: shell,
    ...extra,
  }
}

function runHelper({
  client,
  responsePath,
  home,
  shell = '/bin/sh',
  codexCommand = '',
  env = {},
}) {
  const args = [
    helperPath,
    '--client',
    client,
    '--response',
    responsePath,
    '--home',
    home,
    '--shell',
    shell,
  ]
  if (codexCommand) {
    args.push('--codex-command', codexCommand)
  }
  const result = spawnSync(process.execPath, args, {
    encoding: 'utf8',
    env: helperEnvironment(home, shell, env),
  })
  if (result.error) {
    throw result.error
  }
  return {
    ...result,
    output: `${result.stdout || ''}${result.stderr || ''}`,
  }
}

function backupPaths(filePath) {
  const directory = path.dirname(filePath)
  if (!fs.existsSync(directory)) {
    return []
  }
  const prefix = `${path.basename(filePath)}.tocreate-backup-`
  return fs
    .readdirSync(directory)
    .filter((name) => name.startsWith(prefix))
    .map((name) => path.join(directory, name))
    .sort()
}

function markerCount(content, marker) {
  return content.split(marker).length - 1
}

test('Claude settings are merged through an existing symlink', unixOnly, (t) => {
  const { root, home } = fixture(t)
  const claudeDir = path.join(home, 'claude-config')
  const settingsPath = path.join(claudeDir, 'settings.json')
  const targetPath = path.join(root, 'shared', 'claude-settings.json')
  writeFile(
    targetPath,
    `${JSON.stringify({ theme: 'dark', env: { KEEP_ME: 'yes' } }, null, 2)}\n`,
    0o644,
  )
  fs.mkdirSync(claudeDir, { recursive: true, mode: 0o700 })
  fs.symlinkSync(targetPath, settingsPath)

  const responsePath = writeResponse(root, 'claude-code', {
    api_key: 'sk-claude-secret',
    endpoint: 'https://claude.example.com',
  })
  const options = {
    client: 'claude-code',
    responsePath,
    home,
    env: { CLAUDE_CONFIG_DIR: '~/claude-config' },
  }

  const first = runHelper(options)
  assert.equal(first.status, 0, first.output)
  assert.equal(fs.lstatSync(settingsPath).isSymbolicLink(), true)

  const merged = JSON.parse(fs.readFileSync(targetPath, 'utf8'))
  assert.equal(merged.theme, 'dark')
  assert.equal(merged.env.KEEP_ME, 'yes')
  assert.equal(merged.env.ANTHROPIC_AUTH_TOKEN, 'sk-claude-secret')
  assert.equal(merged.env.ANTHROPIC_BASE_URL, 'https://claude.example.com')
  assert.equal(backupPaths(targetPath).length, 1)

  const firstContent = fs.readFileSync(targetPath, 'utf8')
  const second = runHelper(options)
  assert.equal(second.status, 0, second.output)
  assert.equal(fs.readFileSync(targetPath, 'utf8'), firstContent)
  assert.equal(backupPaths(targetPath).length, 1)
})

test('Gemini environment and settings merges are idempotent', unixOnly, (t) => {
  const { root, home } = fixture(t)
  const geminiDir = path.join(home, 'gemini-home', '.gemini')
  const envPath = path.join(geminiDir, '.env')
  const settingsPath = path.join(geminiDir, 'settings.json')
  writeFile(
    envPath,
    [
      'KEEP_ME=unchanged',
      '',
      '# >>> ToCreate Quick Start >>>',
      'GEMINI_API_KEY="old-key"',
      'GOOGLE_GEMINI_BASE_URL="https://old.example.com"',
      '# <<< ToCreate Quick Start <<<',
      '',
    ].join('\n'),
  )
  writeFile(
    settingsPath,
    `${JSON.stringify({
      theme: 'Default',
      security: { auth: { selectedType: 'oauth-personal' }, keep: true },
    }, null, 2)}\n`,
  )

  const responsePath = writeResponse(root, 'gemini-cli', {
    api_key: 'sk-gemini-secret',
    endpoint: 'https://gemini.example.com',
    model: 'gemini-2.5-pro',
  })
  const options = {
    client: 'gemini-cli',
    responsePath,
    home,
    env: { GEMINI_CLI_HOME: '~/gemini-home' },
  }

  const first = runHelper(options)
  assert.equal(first.status, 0, first.output)
  const mergedEnv = fs.readFileSync(envPath, 'utf8')
  assert.match(mergedEnv, /^KEEP_ME=unchanged/m)
  assert.match(mergedEnv, /^GEMINI_API_KEY="sk-gemini-secret"$/m)
  assert.match(
    mergedEnv,
    /^GOOGLE_GEMINI_BASE_URL="https:\/\/gemini\.example\.com"$/m,
  )
  assert.match(mergedEnv, /^GEMINI_MODEL="gemini-2\.5-pro"$/m)
  assert.equal(markerCount(mergedEnv, '# >>> ToCreate Quick Start >>>'), 1)
  assert.equal(markerCount(mergedEnv, '# <<< ToCreate Quick Start <<<'), 1)

  const settings = JSON.parse(fs.readFileSync(settingsPath, 'utf8'))
  assert.equal(settings.theme, 'Default')
  assert.equal(settings.security.keep, true)
  assert.equal(settings.security.auth.selectedType, 'gemini-api-key')
  assert.equal(backupPaths(envPath).length, 1)
  assert.equal(backupPaths(settingsPath).length, 1)

  const firstSettings = fs.readFileSync(settingsPath, 'utf8')
  const second = runHelper(options)
  assert.equal(second.status, 0, second.output)
  assert.equal(fs.readFileSync(envPath, 'utf8'), mergedEnv)
  assert.equal(fs.readFileSync(settingsPath, 'utf8'), firstSettings)
  assert.equal(backupPaths(envPath).length, 1)
  assert.equal(backupPaths(settingsPath).length, 1)
})

test('Codex config merges, validates, and loads a quoted key path', unixOnly, (t) => {
  const { root, home } = fixture(t, "user's home")
  const codexDir = path.join(home, 'custom-codex')
  const configPath = path.join(codexDir, 'config.toml')
  const profilePath = path.join(home, '.profile')
  const fakeCodex = path.join(root, 'fake-codex')
  writeFile(
    fakeCodex,
    [
      '#!/bin/sh',
      '[ "$1" = "features" ] && [ "$2" = "list" ] || exit 64',
      'printf "validated\\n" >> "$HOME/validation.log"',
      'printf "%s\\n" "$CODEX_HOME" > "$HOME/codex-home.log"',
      '',
    ].join('\n'),
    0o700,
  )
  writeFile(
    configPath,
    [
      '# Keep this comment.',
      'model_provider = "legacy"',
      'model = "legacy-model"',
      '',
      '[features]',
      'shell_snapshot = true',
      '',
      '[model_providers.other]',
      'name = "Other"',
      'experimental_bearer_token = "keep-me"',
      '',
      '[model_providers.tocreate]',
      'name = "Old ToCreate"',
      'base_url = "https://old.example.com"',
      'experimental_bearer_token = "remove-me"',
      'custom_header = "keep-me"',
      '',
    ].join('\n'),
    0o644,
  )
  writeFile(profilePath, 'export KEEP_PROFILE=1\n', 0o644)

  const responsePath = writeResponse(root, 'codex', {
    provider_name: 'ToCreate Custom',
    api_key: 'sk-codex-secret',
    endpoint: 'https://codex.example.com/v1',
    model: 'gpt-5.5',
  })
  const options = {
    client: 'codex',
    responsePath,
    home,
    codexCommand: fakeCodex,
    env: { CODEX_HOME: '~/custom-codex' },
  }

  const first = runHelper(options)
  assert.equal(first.status, 0, first.output)
  const mergedConfig = fs.readFileSync(configPath, 'utf8')
  assert.ok(
    mergedConfig.indexOf('model_provider = "tocreate"') <
      mergedConfig.indexOf('[features]'),
  )
  assert.match(mergedConfig, /^model = "gpt-5\.5"$/m)
  assert.match(mergedConfig, /^\[model_providers\.tocreate\]$/m)
  assert.match(mergedConfig, /^name = "ToCreate Custom"$/m)
  assert.match(mergedConfig, /^base_url = "https:\/\/codex\.example\.com\/v1"$/m)
  assert.match(mergedConfig, /^wire_api = "responses"$/m)
  assert.match(mergedConfig, /^env_key = "TOCREATE_API_KEY"$/m)
  assert.match(mergedConfig, /^requires_openai_auth = false$/m)
  assert.match(mergedConfig, /^custom_header = "keep-me"$/m)
  assert.equal(markerCount(mergedConfig, 'experimental_bearer_token'), 1)
  assert.match(mergedConfig, /experimental_bearer_token = "keep-me"/)

  const keyPath = path.join(codexDir, 'tocreate.key')
  const envPath = path.join(codexDir, 'tocreate.env')
  assert.equal(fs.readFileSync(keyPath, 'utf8'), 'sk-codex-secret\n')
  assert.equal(fs.statSync(keyPath).mode & 0o777, 0o600)
  assert.equal(fs.readFileSync(path.join(home, 'codex-home.log'), 'utf8').trim(), codexDir)

  const sourceResult = spawnSync(
    '/bin/sh',
    [
      '-c',
      '. "$1"; test "$TOCREATE_API_KEY" = "$2"',
      'sh',
      profilePath,
      'sk-codex-secret',
    ],
    {
      encoding: 'utf8',
      env: helperEnvironment(home, '/bin/sh'),
    },
  )
  assert.equal(
    sourceResult.status,
    0,
    `${sourceResult.stdout || ''}${sourceResult.stderr || ''}`,
  )
  if (process.env.REAL_FISH_COMMAND) {
    const fishResult = spawnSync(
      process.env.REAL_FISH_COMMAND,
      [
        '-c',
        'source $argv[1]; test "$TOCREATE_API_KEY" = "$argv[2]"',
        envPath.replace(/\.env$/, '.fish'),
        'sk-codex-secret',
      ],
      {
        encoding: 'utf8',
        env: helperEnvironment(home, process.env.REAL_FISH_COMMAND),
      },
    )
    assert.equal(
      fishResult.status,
      0,
      `${fishResult.stdout || ''}${fishResult.stderr || ''}`,
    )
  }
  assert.match(fs.readFileSync(profilePath, 'utf8'), /KEEP_PROFILE=1/)
  assert.equal(backupPaths(configPath).length, 1)
  assert.equal(backupPaths(profilePath).length, 1)

  const firstProfile = fs.readFileSync(profilePath, 'utf8')
  const firstEnv = fs.readFileSync(envPath, 'utf8')
  const second = runHelper(options)
  assert.equal(second.status, 0, second.output)
  assert.equal(fs.readFileSync(configPath, 'utf8'), mergedConfig)
  assert.equal(fs.readFileSync(profilePath, 'utf8'), firstProfile)
  assert.equal(fs.readFileSync(envPath, 'utf8'), firstEnv)
  assert.equal(backupPaths(configPath).length, 1)
  assert.equal(backupPaths(profilePath).length, 1)
  assert.equal(
    fs.readFileSync(path.join(home, 'validation.log'), 'utf8'),
    'validated\nvalidated\n',
  )
})

test(
  'generated Codex config passes the real Codex validator',
  {
    skip:
      process.platform === 'win32' || !process.env.REAL_CODEX_COMMAND,
  },
  (t) => {
    const { root, home } = fixture(t)
    const configPath = path.join(home, '.codex', 'config.toml')
    writeFile(
      configPath,
      [
        '# Existing user settings remain in place.',
        'model = "legacy-model"',
        '',
        '[features]',
        'shell_snapshot = true',
        '',
      ].join('\n'),
      0o644,
    )
    const responsePath = writeResponse(root, 'codex', {
      api_key: 'sk-real-validator-secret',
      endpoint: 'https://codex.example.com/v1',
      model: 'gpt-5.5',
    })

    const result = runHelper({
      client: 'codex',
      responsePath,
      home,
      codexCommand: process.env.REAL_CODEX_COMMAND,
    })
    assert.equal(result.status, 0, result.output)
    assert.match(
      fs.readFileSync(configPath, 'utf8'),
      /^\[model_providers\.tocreate\]$/m,
    )
  },
)

test('Codex validation failure restores every modified file', unixOnly, (t) => {
  const { root, home } = fixture(t)
  const codexDir = path.join(home, '.codex')
  const configPath = path.join(codexDir, 'config.toml')
  const profilePath = path.join(home, '.profile')
  const fakeCodex = path.join(root, 'reject-codex')
  const originalConfig = [
    'model_provider = "legacy"',
    'model = "legacy-model"',
    '',
    '[features]',
    'shell_snapshot = true',
    '',
  ].join('\n')
  const originalProfile = 'export ORIGINAL_PROFILE=1\n'
  writeFile(configPath, originalConfig, 0o644)
  writeFile(profilePath, originalProfile, 0o644)
  writeFile(
    fakeCodex,
    [
      '#!/bin/sh',
      'printf "%s\\n" "$TOCREATE_API_KEY" >&2',
      'exit 9',
      '',
    ].join('\n'),
    0o700,
  )

  const responsePath = writeResponse(root, 'codex', {
    api_key: 'sk-rollback-secret',
    model: 'gpt-5.5',
  })
  const result = runHelper({
    client: 'codex',
    responsePath,
    home,
    codexCommand: fakeCodex,
  })
  assert.notEqual(result.status, 0)
  assert.match(result.output, /Codex rejected the merged config/)
  assert.match(result.output, /\[redacted\]/)
  assert.doesNotMatch(result.output, /sk-rollback-secret/)
  assert.equal(fs.readFileSync(configPath, 'utf8'), originalConfig)
  assert.equal(fs.readFileSync(profilePath, 'utf8'), originalProfile)
  assert.equal(fs.existsSync(path.join(codexDir, 'tocreate.key')), false)
  assert.equal(fs.existsSync(path.join(codexDir, 'tocreate.env')), false)
  assert.equal(fs.existsSync(path.join(codexDir, 'tocreate.fish')), false)
  assert.equal(backupPaths(configPath).length, 1)
  assert.equal(backupPaths(profilePath).length, 1)
  assert.equal(
    fs.readFileSync(backupPaths(configPath)[0], 'utf8'),
    originalConfig,
  )
})

test('Codex multiline TOML values are preserved during merge', unixOnly, (t) => {
  const { root, home } = fixture(t)
  const codexDir = path.join(home, '.codex')
  const configPath = path.join(codexDir, 'config.toml')
  const originalConfig = [
    'developer_instructions = """',
    'model = "This is user text, not a config key."',
    '"""',
    '',
    '[mcp_servers.demo]',
    'command = "printf"',
    'args = [',
    '  "model = still not a config key",',
    '  "[model_providers.tocreate]",',
    ']',
    '',
  ].join('\n')
  writeFile(configPath, originalConfig, 0o644)
  const responsePath = writeResponse(root, 'codex', {
    api_key: 'sk-multiline-secret',
    model: 'gpt-5.5',
  })

  const result = runHelper({
    client: 'codex',
    responsePath,
    home,
    codexCommand: process.env.REAL_CODEX_COMMAND || '/bin/true',
  })
  assert.equal(result.status, 0, result.output)
  const mergedConfig = fs.readFileSync(configPath, 'utf8')
  assert.match(
    mergedConfig,
    /developer_instructions = """\nmodel = "This is user text, not a config key\."\n"""/,
  )
  assert.match(
    mergedConfig,
    /args = \[\n  "model = still not a config key",\n  "\[model_providers\.tocreate\]",\n\]/,
  )
  assert.match(mergedConfig, /^model_provider = "tocreate"$/m)
  assert.match(mergedConfig, /^model = "gpt-5\.5"$/m)
  assert.equal(backupPaths(configPath).length, 1)
})

test('Codex invalid multiline TOML is refused before any write', unixOnly, (t) => {
  const { root, home } = fixture(t)
  const codexDir = path.join(home, '.codex')
  const configPath = path.join(codexDir, 'config.toml')
  const originalConfig = [
    'developer_instructions = """',
    'This string never closes.',
    '',
  ].join('\n')
  writeFile(configPath, originalConfig, 0o644)
  const responsePath = writeResponse(root, 'codex', {
    api_key: 'sk-invalid-multiline-secret',
    model: 'gpt-5.5',
  })

  const result = runHelper({
    client: 'codex',
    responsePath,
    home,
    codexCommand: '/bin/true',
  })
  assert.notEqual(result.status, 0)
  assert.match(result.output, /unterminated TOML value starting at line 1/)
  assert.equal(fs.readFileSync(configPath, 'utf8'), originalConfig)
  assert.equal(backupPaths(configPath).length, 0)
  assert.equal(fs.existsSync(path.join(codexDir, 'tocreate.key')), false)
})
