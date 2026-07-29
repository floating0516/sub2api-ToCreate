#!/usr/bin/env node

'use strict'

const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const { spawnSync } = require('node:child_process')

const CODEX_PROVIDER_ID = 'tocreate'
const CODEX_ENV_KEY = 'TOCREATE_API_KEY'
const CODEX_PROFILE_START = '# >>> ToCreate Codex environment >>>'
const CODEX_PROFILE_END = '# <<< ToCreate Codex environment <<<'
const GEMINI_ENV_START = '# >>> ToCreate Quick Start >>>'
const GEMINI_ENV_END = '# <<< ToCreate Quick Start <<<'

function fail(message) {
  process.stderr.write(`✗ ${message}\n`)
  process.exit(1)
}

function parseArgs(argv) {
  const options = {}
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index]
    switch (arg) {
      case '--client':
      case '--response':
      case '--home':
      case '--shell':
      case '--codex-command':
        if (index + 1 >= argv.length) {
          throw new Error(`Missing value for ${arg}`)
        }
        options[arg.slice(2).replaceAll('-', '_')] = argv[index + 1]
        index += 1
        break
      case '-h':
      case '--help':
        process.stdout.write(
          'Usage: install-config.js --client <client> --response <redeem.json> --home <home> [--shell <path>]\n',
        )
        process.exit(0)
        break
      default:
        throw new Error(`Unknown argument: ${arg}`)
    }
  }
  return options
}

function isObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function requireString(value, label) {
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`ToCreate response is missing ${label}`)
  }
  if (value.includes('\0') || value.includes('\n') || value.includes('\r')) {
    throw new Error(`ToCreate response contains an invalid ${label}`)
  }
  return value.trim()
}

function readRedeemData(responsePath, expectedClient) {
  let parsed
  try {
    parsed = JSON.parse(fs.readFileSync(responsePath, 'utf8'))
  } catch (error) {
    throw new Error(`Could not read the ToCreate redeem response: ${error.message}`)
  }

  const data = isObject(parsed) && isObject(parsed.data) ? parsed.data : parsed
  if (!isObject(data)) {
    throw new Error('ToCreate redeem response has an invalid shape')
  }

  const client = requireString(data.client, 'client')
  if (client !== expectedClient) {
    throw new Error(`ToCreate response is for ${client}, not ${expectedClient}`)
  }

  return {
    client,
    providerName: requireString(data.provider_name, 'provider name'),
    endpoint: requireString(data.endpoint, 'endpoint'),
    apiKey: requireString(data.api_key, 'API key'),
    model: typeof data.model === 'string' ? data.model.trim() : '',
  }
}

function expandUserPath(raw, home) {
  const value = String(raw || '').trim()
  if (value === '~') {
    return home
  }
  if (value.startsWith(`~${path.sep}`) || value.startsWith('~/')) {
    return path.join(home, value.slice(2))
  }
  return path.resolve(value)
}

function configRoot(envName, fallback, home) {
  const configured = process.env[envName]
  return configured && configured.trim()
    ? expandUserPath(configured, home)
    : fallback
}

function displayPath(filePath, home) {
  if (filePath === home) {
    return '~'
  }
  if (filePath.startsWith(`${home}${path.sep}`)) {
    return `~${path.sep}${filePath.slice(home.length + 1)}`
  }
  return filePath
}

function resolveWriteTarget(requestedPath) {
  if (!fs.existsSync(requestedPath)) {
    return requestedPath
  }

  const stat = fs.lstatSync(requestedPath)
  if (stat.isSymbolicLink()) {
    let resolved
    try {
      resolved = fs.realpathSync(requestedPath)
    } catch {
      throw new Error(`Refusing to replace dangling symlink ${requestedPath}`)
    }
    const resolvedStat = fs.statSync(resolved)
    if (!resolvedStat.isFile()) {
      throw new Error(`Expected ${requestedPath} to point to a regular file`)
    }
    return resolved
  }
  if (!stat.isFile()) {
    throw new Error(`Expected ${requestedPath} to be a regular file`)
  }
  return requestedPath
}

function readOptionalText(requestedPath) {
  const target = resolveWriteTarget(requestedPath)
  if (!fs.existsSync(target)) {
    return { requestedPath, target, original: null, originalMode: null }
  }
  const stat = fs.statSync(target)
  if (!stat.isFile()) {
    throw new Error(`Expected ${requestedPath} to be a regular file`)
  }
  return {
    requestedPath,
    target,
    original: fs.readFileSync(target),
    originalMode: stat.mode & 0o777,
  }
}

function readJsonObject(requestedPath) {
  const existing = readOptionalText(requestedPath)
  if (existing.original === null) {
    return { existing, value: {} }
  }

  let value
  try {
    value = JSON.parse(existing.original.toString('utf8'))
  } catch (error) {
    throw new Error(
      `${requestedPath} is not valid JSON; fix it before rerunning the installer (${error.message})`,
    )
  }
  if (!isObject(value)) {
    throw new Error(`${requestedPath} must contain a JSON object`)
  }
  return { existing, value }
}

function planTextWrite(plans, existing, content, label, mode = 0o600) {
  const bytes = Buffer.from(content, 'utf8')
  if (existing.original !== null && existing.original.equals(bytes)) {
    return
  }
  const duplicate = plans.find((plan) => plan.target === existing.target)
  if (duplicate) {
    if (!duplicate.content.equals(bytes)) {
      throw new Error(`Conflicting updates target ${existing.target}`)
    }
    return
  }
  plans.push({
    ...existing,
    content: bytes,
    label,
    mode,
    backupPath: null,
  })
}

function uniqueBackupPath(target, stamp) {
  for (let suffix = 0; suffix < 1000; suffix += 1) {
    const candidate = `${target}.tocreate-backup-${stamp}${suffix === 0 ? '' : `-${suffix}`}`
    if (!fs.existsSync(candidate)) {
      return candidate
    }
  }
  throw new Error(`Could not allocate a backup path for ${target}`)
}

function writeAtomic(target, content, mode) {
  const parent = path.dirname(target)
  fs.mkdirSync(parent, { recursive: true, mode: 0o700 })
  const tempPath = path.join(
    parent,
    `.${path.basename(target)}.tocreate-${process.pid}-${Math.random().toString(16).slice(2)}`,
  )
  try {
    fs.writeFileSync(tempPath, content, { flag: 'wx', mode })
    fs.chmodSync(tempPath, mode)
    fs.renameSync(tempPath, target)
  } finally {
    try {
      fs.unlinkSync(tempPath)
    } catch {
      // The rename succeeded or the temporary file was never created.
    }
  }
}

function rollbackPlans(applied) {
  for (const plan of [...applied].reverse()) {
    try {
      if (plan.original === null) {
        fs.unlinkSync(plan.target)
      } else {
        writeAtomic(plan.target, plan.original, plan.originalMode || plan.mode)
      }
    } catch (error) {
      process.stderr.write(`! Could not roll back ${plan.target}: ${error.message}\n`)
    }
  }
}

function applyPlans(plans, validate) {
  if (plans.length === 0) {
    if (validate) {
      validate()
    }
    return []
  }

  const stamp = new Date()
    .toISOString()
    .replace(/[-:]/g, '')
    .replace(/\.\d{3}Z$/, 'Z')
  const createdBackups = []
  try {
    for (const plan of plans) {
      if (plan.original !== null) {
        plan.backupPath = uniqueBackupPath(plan.target, stamp)
        fs.copyFileSync(plan.target, plan.backupPath, fs.constants.COPYFILE_EXCL)
        createdBackups.push(plan)
        fs.chmodSync(plan.backupPath, 0o600)
      }
    }
  } catch (error) {
    for (const plan of createdBackups.reverse()) {
      try {
        fs.unlinkSync(plan.backupPath)
      } catch {
        // Leave the original backup error as the actionable failure.
      }
      plan.backupPath = null
    }
    throw error
  }

  const applied = []
  try {
    for (const plan of plans) {
      writeAtomic(plan.target, plan.content, plan.mode)
      applied.push(plan)
    }
    if (validate) {
      validate()
    }
  } catch (error) {
    rollbackPlans(applied)
    throw error
  }
  return plans
}

function printPlanResult(plans, home) {
  if (plans.length === 0) {
    process.stdout.write('✓ Existing configuration already matches ToCreate\n')
    return
  }
  for (const plan of plans) {
    if (plan.backupPath) {
      process.stdout.write(
        `  Backup: ${displayPath(plan.backupPath, home)}\n`,
      )
    }
    process.stdout.write(
      `✓ ${plan.label}: ${displayPath(plan.requestedPath, home)}\n`,
    )
  }
}

function configureClaude(data, home) {
  const claudeDir = configRoot(
    'CLAUDE_CONFIG_DIR',
    path.join(home, '.claude'),
    home,
  )
  const settingsPath = path.join(claudeDir, 'settings.json')
  const { existing, value } = readJsonObject(settingsPath)
  if (value.env !== undefined && !isObject(value.env)) {
    throw new Error(`${settingsPath} has a non-object "env" field`)
  }
  value.env = {
    ...(value.env || {}),
    ANTHROPIC_AUTH_TOKEN: data.apiKey,
    ANTHROPIC_BASE_URL: data.endpoint,
  }

  const plans = []
  planTextWrite(
    plans,
    existing,
    `${JSON.stringify(value, null, 2)}\n`,
    'Claude Code settings merged',
  )
  const applied = applyPlans(plans)
  printPlanResult(applied, home)
}

function envAssignmentKey(line) {
  const match = line.match(/^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=(.*)$/)
  if (!match) {
    return null
  }
  const rawValue = match[2].trim()
  if (
    rawValue.endsWith('\\') ||
    ((rawValue.startsWith('"') || rawValue.startsWith("'")) &&
      rawValue.at(-1) !== rawValue[0])
  ) {
    throw new Error(
      `Cannot safely replace multiline .env assignment for ${match[1]}`,
    )
  }
  return match[1]
}

function quoteEnvValue(value) {
  return JSON.stringify(value)
}

function mergeGeminiEnv(content, entries) {
  const newline = content.includes('\r\n') ? '\r\n' : '\n'
  const lines = content === '' ? [] : content.split(/\r?\n/)
  if (lines.at(-1) === '') {
    lines.pop()
  }

  const keys = new Set(Object.keys(entries))
  const kept = []
  for (const line of lines) {
    if (line.trim() === GEMINI_ENV_START || line.trim() === GEMINI_ENV_END) {
      continue
    }
    const key = envAssignmentKey(line)
    if (key && keys.has(key)) {
      continue
    }
    kept.push(line)
  }

  while (kept.length > 0 && kept.at(-1).trim() === '') {
    kept.pop()
  }
  if (kept.length > 0) {
    kept.push('')
  }
  kept.push(GEMINI_ENV_START)
  for (const [key, value] of Object.entries(entries)) {
    kept.push(`${key}=${quoteEnvValue(value)}`)
  }
  kept.push(GEMINI_ENV_END)
  return `${kept.join(newline)}${newline}`
}

function configureGemini(data, home) {
  const geminiHome = configRoot('GEMINI_CLI_HOME', home, home)
  const geminiDir = path.join(geminiHome, '.gemini')
  const envPath = path.join(geminiDir, '.env')
  const settingsPath = path.join(geminiDir, 'settings.json')

  const envExisting = readOptionalText(envPath)
  const envEntries = {
    GEMINI_API_KEY: data.apiKey,
    GOOGLE_GEMINI_BASE_URL: data.endpoint,
  }
  if (data.model) {
    envEntries.GEMINI_MODEL = data.model
  }
  const mergedEnv = mergeGeminiEnv(
    envExisting.original ? envExisting.original.toString('utf8') : '',
    envEntries,
  )

  const { existing: settingsExisting, value: settings } =
    readJsonObject(settingsPath)
  if (settings.security !== undefined && !isObject(settings.security)) {
    throw new Error(`${settingsPath} has a non-object "security" field`)
  }
  settings.security = settings.security || {}
  if (settings.security.auth !== undefined && !isObject(settings.security.auth)) {
    throw new Error(`${settingsPath} has a non-object "security.auth" field`)
  }
  settings.security.auth = settings.security.auth || {}
  settings.security.auth.selectedType = 'gemini-api-key'

  const plans = []
  planTextWrite(plans, envExisting, mergedEnv, 'Gemini CLI environment merged')
  planTextWrite(
    plans,
    settingsExisting,
    `${JSON.stringify(settings, null, 2)}\n`,
    'Gemini CLI settings merged',
  )
  const applied = applyPlans(plans)
  printPlanResult(applied, home)
}

function tomlString(value) {
  return JSON.stringify(value)
}

function splitTomlPath(raw) {
  const parts = []
  let current = ''
  let quote = null
  let escaped = false

  for (const char of raw.trim()) {
    if (quote) {
      current += char
      if (quote === '"' && escaped) {
        escaped = false
      } else if (quote === '"' && char === '\\') {
        escaped = true
      } else if (char === quote) {
        quote = null
      }
      continue
    }
    if (char === '"' || char === "'") {
      quote = char
      current += char
    } else if (char === '.') {
      parts.push(current.trim())
      current = ''
    } else {
      current += char
    }
  }
  if (quote) {
    throw new Error(`Unterminated quoted TOML key: ${raw}`)
  }
  parts.push(current.trim())

  return parts.map((part) => {
    if (/^[A-Za-z0-9_-]+$/.test(part)) {
      return part
    }
    if (part.startsWith('"') && part.endsWith('"')) {
      try {
        return JSON.parse(part)
      } catch {
        throw new Error(`Invalid quoted TOML key: ${part}`)
      }
    }
    if (part.startsWith("'") && part.endsWith("'")) {
      return part.slice(1, -1)
    }
    throw new Error(`Unsupported TOML key syntax: ${part}`)
  })
}

function parseTomlHeader(line) {
  const trimmed = line.trim()
  if (!trimmed.startsWith('[')) {
    return null
  }
  const arrayTable = trimmed.startsWith('[[')
  const openLength = arrayTable ? 2 : 1
  const close = arrayTable ? ']]' : ']'
  let quote = null
  let escaped = false

  for (let index = openLength; index < trimmed.length; index += 1) {
    const char = trimmed[index]
    if (quote) {
      if (quote === '"' && escaped) {
        escaped = false
      } else if (quote === '"' && char === '\\') {
        escaped = true
      } else if (char === quote) {
        quote = null
      }
      continue
    }
    if (char === '"' || char === "'") {
      quote = char
      continue
    }
    if (trimmed.startsWith(close, index)) {
      const rest = trimmed.slice(index + close.length).trim()
      if (rest !== '' && !rest.startsWith('#')) {
        return null
      }
      return {
        arrayTable,
        path: splitTomlPath(trimmed.slice(openLength, index)),
      }
    }
  }
  return null
}

function findTomlAssignment(line) {
  let quote = null
  let escaped = false
  for (let index = 0; index < line.length; index += 1) {
    const char = line[index]
    if (quote) {
      if (quote === '"' && escaped) {
        escaped = false
      } else if (quote === '"' && char === '\\') {
        escaped = true
      } else if (char === quote) {
        quote = null
      }
      continue
    }
    if (char === '"' || char === "'") {
      quote = char
    } else if (char === '#') {
      return null
    } else if (char === '=') {
      return {
        path: splitTomlPath(line.slice(0, index)),
        value: line.slice(index + 1).trim(),
      }
    }
  }
  return null
}

function scalarTomlAssignment(name, value) {
  return `${name} = ${tomlString(value)}`
}

function ensureSingleLineTomlValue(value, label) {
  const trimmed = value.trim()
  if (
    trimmed.startsWith('"""') ||
    trimmed.startsWith("'''") ||
    trimmed.startsWith('[') ||
    trimmed.startsWith('{')
  ) {
    throw new Error(`Cannot safely merge multiline or structured ${label}`)
  }
}

function scanTomlValueFragment(fragment, state, lineNumber) {
  const current = state || {
    stack: [],
    quote: null,
    multiline: null,
    escaped: false,
    sawValue: false,
    startingLine: lineNumber,
  }

  for (let index = 0; index < fragment.length; index += 1) {
    const char = fragment[index]
    if (current.multiline) {
      if (current.multiline === '"""' && current.escaped) {
        current.escaped = false
      } else if (current.multiline === '"""' && char === '\\') {
        current.escaped = true
      } else if (fragment.startsWith(current.multiline, index)) {
        index += current.multiline.length - 1
        current.multiline = null
      }
      continue
    }
    if (current.quote) {
      if (current.quote === '"' && current.escaped) {
        current.escaped = false
      } else if (current.quote === '"' && char === '\\') {
        current.escaped = true
      } else if (char === current.quote) {
        current.quote = null
      }
      continue
    }
    if (char === '#') {
      break
    }
    if (fragment.startsWith('"""', index)) {
      current.multiline = '"""'
      current.sawValue = true
      index += 2
    } else if (fragment.startsWith("'''", index)) {
      current.multiline = "'''"
      current.sawValue = true
      index += 2
    } else if (char === '"' || char === "'") {
      current.quote = char
      current.sawValue = true
    } else if (char === '[') {
      current.stack.push(']')
      current.sawValue = true
    } else if (char === '{') {
      current.stack.push('}')
      current.sawValue = true
    } else if (char === ']' || char === '}') {
      if (current.stack.pop() !== char) {
        throw new Error(`Codex config has invalid TOML at line ${lineNumber}`)
      }
    } else if (!/\s/.test(char)) {
      current.sawValue = true
    }
  }

  if (current.quote) {
    throw new Error(
      `Codex config has an unterminated TOML string at line ${lineNumber}`,
    )
  }
  if (current.multiline === '"""' && current.escaped) {
    current.escaped = false
  }
  if (!current.sawValue) {
    throw new Error(`Codex config has an empty TOML value at line ${lineNumber}`)
  }
  if (current.multiline || current.stack.length > 0) {
    return current
  }
  return null
}

function sameTomlPath(left, right) {
  return left.length === right.length &&
    left.every((part, index) => part === right[index])
}

function isTomlPrefix(prefix, value) {
  return prefix.length <= value.length &&
    prefix.every((part, index) => part === value[index])
}

function mergeCodexToml(content, data) {
  const newline = content.includes('\r\n') ? '\r\n' : '\n'
  const lines = content === '' ? [] : content.split(/\r?\n/)
  if (lines.at(-1) === '') {
    lines.pop()
  }

  const providerPath = ['model_providers', CODEX_PROVIDER_ID]
  const rootValues = {
    model_provider: CODEX_PROVIDER_ID,
    model: data.model || 'gpt-5.5',
  }
  const providerValues = {
    name: data.providerName,
    base_url: data.endpoint,
    wire_api: 'responses',
    env_key: CODEX_ENV_KEY,
    requires_openai_auth: false,
  }

  let currentTable = []
  let firstHeader = lines.length
  let providerHeader = -1
  let providerEnd = lines.length
  let activeProviderTable = false
  let tomlValueState = null
  const rootAssignments = new Map()
  const providerAssignments = new Map()
  const replacements = new Map()

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index]
    if (tomlValueState) {
      tomlValueState = scanTomlValueFragment(
        line,
        tomlValueState,
        index + 1,
      )
      continue
    }
    const header = parseTomlHeader(line)
    if (header) {
      if (firstHeader === lines.length) {
        firstHeader = index
      }
      if (activeProviderTable && providerEnd === lines.length) {
        providerEnd = index
      }
      currentTable = header.path
      activeProviderTable = !header.arrayTable &&
        sameTomlPath(currentTable, providerPath)
      if (activeProviderTable) {
        if (providerHeader !== -1) {
          throw new Error(
            `Duplicate [model_providers.${CODEX_PROVIDER_ID}] tables in Codex config`,
          )
        }
        providerHeader = index
      } else if (
        isTomlPrefix(providerPath, currentTable) &&
        providerHeader === -1
      ) {
        throw new Error(
          `Codex config defines a child of [model_providers.${CODEX_PROVIDER_ID}] without the provider table`,
        )
      }
      continue
    }

    const assignment = findTomlAssignment(line)
    if (!assignment) {
      continue
    }
    tomlValueState = scanTomlValueFragment(
      assignment.value,
      null,
      index + 1,
    )

    if (currentTable.length === 0) {
      if (
        assignment.path[0] === 'model_providers' &&
        assignment.path.length >= 1
      ) {
        throw new Error(
          'Codex config uses inline or dotted model_providers syntax that cannot be merged safely',
        )
      }
      if (assignment.path.length === 1 && rootValues[assignment.path[0]] !== undefined) {
        const key = assignment.path[0]
        if (rootAssignments.has(key)) {
          throw new Error(`Duplicate top-level ${key} entries in Codex config`)
        }
        ensureSingleLineTomlValue(assignment.value, `Codex ${key}`)
        rootAssignments.set(key, index)
        replacements.set(
          index,
          typeof rootValues[key] === 'boolean'
            ? `${key} = ${rootValues[key]}`
            : scalarTomlAssignment(key, rootValues[key]),
        )
      }
      continue
    }

    if (!sameTomlPath(currentTable, providerPath) || assignment.path.length !== 1) {
      continue
    }
    const key = assignment.path[0]
    if (key === 'auth') {
      throw new Error(
        `Codex provider ${CODEX_PROVIDER_ID} has command-backed auth that cannot be replaced safely`,
      )
    }
    if (key === 'experimental_bearer_token') {
      ensureSingleLineTomlValue(assignment.value, 'Codex bearer token')
      replacements.set(index, null)
      continue
    }
    if (providerValues[key] === undefined) {
      continue
    }
    if (providerAssignments.has(key)) {
      throw new Error(
        `Duplicate ${key} entries in [model_providers.${CODEX_PROVIDER_ID}]`,
      )
    }
    ensureSingleLineTomlValue(assignment.value, `Codex provider ${key}`)
    providerAssignments.set(key, index)
      replacements.set(
      index,
      typeof providerValues[key] === 'boolean'
        ? `${key} = ${providerValues[key]}`
        : scalarTomlAssignment(key, providerValues[key]),
    )
  }

  if (tomlValueState) {
    throw new Error(
      `Codex config has an unterminated TOML value starting at line ${tomlValueState.startingLine}`,
    )
  }

  if (activeProviderTable && providerEnd === lines.length) {
    providerEnd = lines.length
  }

  const missingRoot = Object.keys(rootValues).filter(
    (key) => !rootAssignments.has(key),
  )
  const missingProvider = Object.keys(providerValues).filter(
    (key) => !providerAssignments.has(key),
  )
  const output = []

  function appendBlankIfNeeded() {
    if (output.length > 0 && output.at(-1).trim() !== '') {
      output.push('')
    }
  }

  for (let index = 0; index <= lines.length; index += 1) {
    if (index === firstHeader && missingRoot.length > 0) {
      appendBlankIfNeeded()
      for (const key of missingRoot) {
        output.push(scalarTomlAssignment(key, rootValues[key]))
      }
      if (index < lines.length) {
        output.push('')
      }
    }
    if (
      providerHeader !== -1 &&
      index === providerEnd &&
      missingProvider.length > 0
    ) {
      for (const key of missingProvider) {
        output.push(
          typeof providerValues[key] === 'boolean'
            ? `${key} = ${providerValues[key]}`
            : scalarTomlAssignment(key, providerValues[key]),
        )
      }
    }
    if (index === lines.length) {
      break
    }
    if (replacements.has(index)) {
      const replacement = replacements.get(index)
      if (replacement !== null) {
        output.push(replacement)
      }
    } else {
      output.push(lines[index])
    }
  }

  if (providerHeader === -1) {
    appendBlankIfNeeded()
    output.push(`[model_providers.${CODEX_PROVIDER_ID}]`)
    for (const [key, value] of Object.entries(providerValues)) {
      output.push(
        typeof value === 'boolean'
          ? `${key} = ${value}`
          : scalarTomlAssignment(key, value),
      )
    }
  }

  return `${output.join(newline).replace(/\s+$/, '')}${newline}`
}

function shellSingleQuote(value) {
  return `'${value.replaceAll("'", "'\"'\"'")}'`
}

function fishSingleQuote(value) {
  return `'${value
    .replaceAll('\\', '\\\\')
    .replaceAll("'", "\\'")}'`
}

function replaceManagedBlock(content, block) {
  const start = content.indexOf(CODEX_PROFILE_START)
  const end = content.indexOf(CODEX_PROFILE_END)
  if ((start === -1) !== (end === -1) || (start !== -1 && end < start)) {
    throw new Error('A shell profile contains an incomplete ToCreate managed block')
  }

  let base = content
  if (start !== -1) {
    base = `${content.slice(0, start)}${content.slice(end + CODEX_PROFILE_END.length)}`
  }
  base = base.replace(/\s+$/, '')
  return `${base}${base ? os.EOL.repeat(2) : ''}${block}${os.EOL}`
}

function shellProfilePaths(home, shellPath) {
  const shell = path.basename(shellPath || '')
  if (shell === 'fish') {
    return {
      shell,
      paths: [path.join(home, '.config', 'fish', 'conf.d', 'tocreate.fish')],
      warning: '',
    }
  }
  if (shell === 'zsh') {
    return {
      shell,
      paths: [path.join(home, '.zshrc'), path.join(home, '.zprofile')],
      warning: '',
    }
  }
  if (shell === 'bash') {
    const loginCandidates = [
      path.join(home, '.bash_profile'),
      path.join(home, '.bash_login'),
      path.join(home, '.profile'),
    ]
    const loginProfile =
      loginCandidates.find((candidate) => fs.existsSync(candidate)) ||
      path.join(home, '.profile')
    return {
      shell,
      paths: [path.join(home, '.bashrc'), loginProfile],
      warning: '',
    }
  }
  if (['sh', 'dash', 'ksh', 'mksh'].includes(shell)) {
    return {
      shell,
      paths: [path.join(home, '.profile')],
      warning: '',
    }
  }
  return {
    shell,
    paths: [path.join(home, '.profile')],
    warning:
      'Unknown login shell; added the Codex environment loader to ~/.profile',
  }
}

function validateCodexConfig({
  codexCommand,
  codexDir,
  home,
  apiKey,
}) {
  const result = spawnSync(codexCommand || 'codex', ['features', 'list'], {
    encoding: 'utf8',
    timeout: 30000,
    env: {
      ...process.env,
      HOME: home,
      USERPROFILE: home,
      CODEX_HOME: codexDir,
      [CODEX_ENV_KEY]: apiKey,
    },
  })
  if (result.error) {
    throw new Error(`Could not validate Codex config: ${result.error.message}`)
  }
  if (result.status !== 0) {
    const details = `${result.stderr || result.stdout || ''}`
      .replaceAll(apiKey, '[redacted]')
      .trim()
      .split(/\r?\n/)
      .slice(-8)
      .join('\n')
    throw new Error(
      `Codex rejected the merged config${details ? `:\n${details}` : ''}`,
    )
  }
}

function configureCodex(data, home, shellPath, codexCommand) {
  const codexDir = configRoot(
    'CODEX_HOME',
    path.join(home, '.codex'),
    home,
  )
  const configPath = path.join(codexDir, 'config.toml')
  const keyPath = path.join(codexDir, 'tocreate.key')
  const envPath = path.join(codexDir, 'tocreate.env')
  const fishEnvPath = path.join(codexDir, 'tocreate.fish')

  const configExisting = readOptionalText(configPath)
  const mergedConfig = mergeCodexToml(
    configExisting.original ? configExisting.original.toString('utf8') : '',
    data,
  )

  const keyExisting = readOptionalText(keyPath)
  const envExisting = readOptionalText(envPath)
  const fishEnvExisting = readOptionalText(fishEnvPath)
  const quotedKeyPath = shellSingleQuote(keyPath)
  const envContent = [
    '# Managed by ToCreate Quick Start. Keep tocreate.key private.',
    `if [ -r ${quotedKeyPath} ]; then`,
    `  ${CODEX_ENV_KEY}=$(cat ${quotedKeyPath})`,
    `  export ${CODEX_ENV_KEY}`,
    'fi',
    '',
  ].join('\n')
  const fishContent = [
    '# Managed by ToCreate Quick Start. Keep tocreate.key private.',
    `if test -r ${fishSingleQuote(keyPath)}`,
    `  set -gx ${CODEX_ENV_KEY} (string collect < ${fishSingleQuote(keyPath)})`,
    'end',
    '',
  ].join('\n')

  const profileInfo = shellProfilePaths(home, shellPath)
  const plans = []
  planTextWrite(
    plans,
    configExisting,
    mergedConfig,
    'Codex provider merged',
  )
  planTextWrite(
    plans,
    keyExisting,
    `${data.apiKey}\n`,
    'Codex API key saved',
  )
  planTextWrite(
    plans,
    envExisting,
    envContent,
    'Codex environment loader saved',
  )
  planTextWrite(
    plans,
    fishEnvExisting,
    fishContent,
    'Codex fish environment loader saved',
  )

  for (const profilePath of profileInfo.paths) {
    const existing = readOptionalText(profilePath)
    const isFishProfile = profileInfo.shell === 'fish'
    const loader = isFishProfile
      ? `source ${fishSingleQuote(fishEnvPath)}`
      : `[ -r ${shellSingleQuote(envPath)} ] && . ${shellSingleQuote(envPath)}`
    const block = `${CODEX_PROFILE_START}${os.EOL}${loader}${os.EOL}${CODEX_PROFILE_END}`
    const merged = replaceManagedBlock(
      existing.original ? existing.original.toString('utf8') : '',
      block,
    )
    planTextWrite(
      plans,
      existing,
      merged,
      'Shell profile merged',
      existing.originalMode || 0o600,
    )
  }

  const applied = applyPlans(plans, () =>
    validateCodexConfig({
      codexCommand,
      codexDir,
      home,
      apiKey: data.apiKey,
    }),
  )
  printPlanResult(applied, home)
  if (profileInfo.warning) {
    process.stdout.write(`! ${profileInfo.warning}\n`)
  }
}

function main() {
  const options = parseArgs(process.argv.slice(2))
  const client = requireString(options.client, '--client')
  const responsePath = requireString(options.response, '--response')
  const home = path.resolve(requireString(options.home, '--home'))
  const data = readRedeemData(responsePath, client)

  switch (client) {
    case 'claude-code':
      configureClaude(data, home)
      break
    case 'codex':
      configureCodex(
        data,
        home,
        options.shell || process.env.SHELL || '',
        options.codex_command || 'codex',
      )
      break
    case 'gemini-cli':
      configureGemini(data, home)
      break
    default:
      throw new Error(`Unsupported install client: ${client}`)
  }
}

try {
  main()
} catch (error) {
  fail(error instanceof Error ? error.message : String(error))
}
