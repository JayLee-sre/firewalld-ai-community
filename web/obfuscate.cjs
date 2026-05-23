#!/usr/bin/env node
// Post-build JS obfuscation for production
// Run: node obfuscate.cjs

const fs = require('fs')
const path = require('path')
const JavaScriptObfuscator = require('javascript-obfuscator')

const distDir = path.join(__dirname, 'dist', 'assets')

const obfuscationOptions = {
  compact: true,
  controlFlowFlattening: true,
  controlFlowFlatteningThreshold: 0.5,
  deadCodeInjection: true,
  deadCodeInjectionThreshold: 0.2,
  debugProtection: false,
  debugProtectionInterval: 0,
  disableConsoleOutput: false,
  identifierNamesGenerator: 'hexadecimal',
  log: false,
  numbersToExpressions: true,
  renameGlobals: false,
  selfDefending: false,
  simplify: true,
  splitStrings: true,
  splitStringsChunkLength: 15,
  stringArray: true,
  stringArrayEncoding: ['rc4'],
  stringArrayIndexShift: true,
  stringArrayRotate: true,
  stringArrayShuffle: true,
  stringArrayWrappersCount: 1,
  stringArrayWrappersChainedCalls: true,
  stringArrayWrappersParametersMaxCount: 2,
  stringArrayWrappersType: 'function',
  stringArrayThreshold: 0.75,
  transformObjectKeys: true,
  unicodeEscapeSequence: false,
}

function obfuscateFile(filePath) {
  const code = fs.readFileSync(filePath, 'utf-8')
  const result = JavaScriptObfuscator.obfuscate(code, obfuscationOptions)
  fs.writeFileSync(filePath, result.getObfuscatedCode(), 'utf-8')
}

function main() {
  if (!fs.existsSync(distDir)) {
    console.error('dist/assets not found, run vite build first')
    process.exit(1)
  }

  const files = fs.readdirSync(distDir).filter(f => f.endsWith('.js'))
  console.log(`Obfuscating ${files.length} JS files...`)

  for (const file of files) {
    const filePath = path.join(distDir, file)
    const before = fs.statSync(filePath).size
    obfuscateFile(filePath)
    const after = fs.statSync(filePath).size
    console.log(`  ✓ ${file} (${(before / 1024).toFixed(0)}KB → ${(after / 1024).toFixed(0)}KB)`)
  }

  console.log(`Done. ${files.length} files obfuscated.`)
}

main()
