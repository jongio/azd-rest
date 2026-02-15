/**
 * CLI Reference Generator for azd-rest
 *
 * Generates reference pages from the CLI help output at build time.
 * Creates:
 * - /reference/cli/index.astro (overview)
 * - /reference/cli/[command].astro (individual command pages)
 */

import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';
import { execSync } from 'child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const OUTPUT_DIR = path.resolve(__dirname, '../src/pages/reference/cli');

interface CommandInfo {
  name: string;
  description: string;
  usage: string;
  flags: Flag[];
}

interface Flag {
  flag: string;
  short: string;
  description: string;
}

/**
 * Parse the help output from a command
 */
function parseHelpOutput(helpText: string): { description: string; flags: Flag[] } {
  const lines = helpText.split('\n');
  let description = '';
  const flags: Flag[] = [];

  // First non-empty line that isn't a section header is the description
  for (const line of lines) {
    if (line.trim() && !line.startsWith('Usage:') && !line.startsWith('Available Commands:') && !line.startsWith('Flags:')) {
      description = line.trim();
      break;
    }
  }

  // Parse flags section
  let inFlags = false;
  for (const line of lines) {
    if (line.startsWith('Flags:') || line.startsWith('Global Flags:')) {
      inFlags = true;
      continue;
    }
    if (line.startsWith('Use "') || (inFlags && line.trim() === '' && flags.length > 0)) {
      inFlags = false;
      continue;
    }
    if (inFlags && line.trim()) {
      // Parse flag line like: "  -h, --help   help for version"
      const match = line.match(/^\s+(-\w)?,?\s*(--[\w-]+)\s+(.+)$/);
      if (match) {
        flags.push({
          short: match[1] || '',
          flag: match[2],
          description: match[3].trim()
        });
      } else {
        // Try matching just long flag
        const longMatch = line.match(/^\s+(--[\w-]+)\s+(.+)$/);
        if (longMatch) {
          flags.push({
            short: '',
            flag: longMatch[1],
            description: longMatch[2].trim()
          });
        }
      }
    }
  }

  return { description, flags };
}

/**
 * Get commands by running the CLI
 */
function discoverCommands(): CommandInfo[] {
  const commands: CommandInfo[] = [];

  try {
    // Get main help
    const mainHelp = execSync('go run ./src/cmd/rest --help 2>&1', {
      encoding: 'utf-8',
      cwd: path.resolve(__dirname, '../../cli')
    });

    // Find available commands
    const commandMatch = mainHelp.match(/Available Commands:\n([\s\S]*?)(?=\n\nFlags:|$)/);
    if (commandMatch) {
      const commandLines = commandMatch[1].split('\n').filter(l => l.trim());

      for (const line of commandLines) {
        const match = line.match(/^\s+(\w+)\s+(.+)$/);
        if (match && !['completion', 'help'].includes(match[1])) {
          const cmdName = match[1];
          const cmdDesc = match[2].trim();

          // Get detailed help for this command
          try {
            const cmdHelp = execSync(`go run ./src/cmd/rest ${cmdName} --help 2>&1`, {
              encoding: 'utf-8',
              cwd: path.resolve(__dirname, '../../cli')
            });

            const parsed = parseHelpOutput(cmdHelp);
            commands.push({
              name: cmdName,
              description: cmdDesc,
              usage: `azd rest ${cmdName} [flags]`,
              flags: parsed.flags
            });
          } catch {
            commands.push({
              name: cmdName,
              description: cmdDesc,
              usage: `azd rest ${cmdName} [flags]`,
              flags: []
            });
          }
        }
      }
    }
  } catch (err) {
    console.warn('⚠️  Could not run CLI to discover commands. Skipping generation to preserve existing pages.');
    return commands;
  }

  return commands;
}

/** Escape curly braces for Astro/JSX template output */
function escapeJsx(text: string): string {
  return text.replace(/\{/g, '&#123;').replace(/\}/g, '&#125;');
}

function generateFlagsTable(command: CommandInfo): string {
  if (command.flags.length === 0) return '';

  const rows = command.flags.map(f =>
    `              <tr>
                <td><code>${f.flag}</code></td>
                <td>${f.short ? `<code>${f.short}</code>` : '-'}</td>
                <td>${escapeJsx(f.description)}</td>
              </tr>`
  ).join('\n');

  return `
        <section class="section">
          <h2>Flags</h2>
          <div class="flags-table">
            <table>
              <thead>
                <tr>
                  <th>Flag</th>
                  <th>Short</th>
                  <th>Description</th>
                </tr>
              </thead>
              <tbody>
${rows}
              </tbody>
            </table>
          </div>
        </section>`;
}

function generateCommandPage(command: CommandInfo): string {
  return `---
import Layout from '../../../components/Layout.astro';
---

<Layout title="${command.name} - CLI Reference" description="Reference for azd rest ${command.name} command">
  <div class="page-container">
    <nav class="breadcrumb">
      <a href={import.meta.env.BASE_URL}>Home</a> /
      <a href={\`\${import.meta.env.BASE_URL}reference/cli/\`}>CLI Reference</a> /
      <span>${command.name}</span>
    </nav>

    <div class="page-header">
      <h1>azd rest ${command.name}</h1>
      <p class="page-intro">${escapeJsx(command.description)}</p>
    </div>

    <div class="content">
      <section class="section">
        <h2>Usage</h2>
        <pre><code>${escapeJsx(command.usage)}</code></pre>
      </section>
${generateFlagsTable(command)}
      <div class="back-link">
        <a href={\`\${import.meta.env.BASE_URL}reference/cli/\`}>&larr; Back to CLI Reference</a>
      </div>
    </div>
  </div>
</Layout>

<style>
  .page-container {
    max-width: 1280px;
    margin: 0 auto;
    padding: 2rem 1.5rem;
  }

  .breadcrumb {
    font-size: 0.875rem;
    color: var(--color-text-secondary);
    margin-bottom: 2rem;
  }

  .breadcrumb a {
    color: var(--color-text-secondary);
  }

  .breadcrumb a:hover {
    color: var(--color-interactive-default);
  }

  .page-header {
    margin-bottom: 3rem;
  }

  .page-header h1 {
    font-size: 2.5rem;
    font-weight: 700;
    margin: 0 0 1rem;
    color: var(--color-text-primary);
  }

  .page-intro {
    font-size: 1.125rem;
    color: var(--color-text-secondary);
  }

  .content {
    max-width: 1280px;
  }

  .section {
    margin-bottom: 3rem;
  }

  .section h2 {
    font-size: 1.75rem;
    font-weight: 600;
    margin: 0 0 1rem;
    color: var(--color-text-primary);
  }

  pre {
    background-color: var(--color-bg-secondary);
    padding: 1rem;
    border-radius: 0.5rem;
    overflow-x: auto;
  }

  .flags-table {
    overflow-x: auto;
    margin: 1.5rem 0;
  }

  .flags-table table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.9375rem;
  }

  .flags-table th {
    background-color: var(--color-bg-secondary);
    padding: 0.75rem;
    text-align: left;
    font-weight: 600;
    color: var(--color-text-primary);
    border-bottom: 2px solid var(--color-border-default);
  }

  .flags-table td {
    padding: 0.75rem;
    border-bottom: 1px solid var(--color-border-default);
    color: var(--color-text-secondary);
  }

  .flags-table code {
    background-color: var(--color-bg-secondary);
    padding: 0.125rem 0.375rem;
    border-radius: 0.25rem;
    font-size: 0.875rem;
  }

  .back-link {
    margin-top: 3rem;
    padding-top: 2rem;
    border-top: 1px solid var(--color-border-default);
  }
</style>
`;
}

function generateIndexPage(commands: CommandInfo[]): string {
  const commandCards = commands.map(cmd => `
          <a href={\`\${base}reference/cli/${cmd.name}/\`} class="command-card">
            <h3><code>rest ${cmd.name}</code></h3>
            <p>${escapeJsx(cmd.description)}</p>
            <span class="command-meta">${cmd.flags.length} flags</span>
          </a>`
  ).join('\n');

  // Build the POST example string carefully - it contains braces and quotes
  const postExampleStr = 'azd rest post https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}?api-version=2021-04-01 \\\\\n  --data \'{"location":"eastus","sku":{"name":"Standard_LRS"},"kind":"StorageV2"}\'';
  // Escape for embedding in a JS string literal in the generated .astro file
  const postExampleEscaped = JSON.stringify(postExampleStr);

  return `---
import Layout from '../../../components/Layout.astro';
import CodeBlock from '../../../components/CodeBlock.astro';

const base = import.meta.env.BASE_URL;
const postExample = ${postExampleEscaped};
---

<Layout title="CLI Reference" description="Complete reference for azd rest command-line interface">
  <div class="page-container">
    <div class="page-header">
      <h1>CLI Reference</h1>
      <p class="page-intro">
        Complete command-line interface documentation for azd rest.
      </p>
    </div>

    <div class="content">
      <section class="section">
        <h2>Overview</h2>
        <p>
          The azd rest extension provides commands to execute REST API calls with automatic Azure authentication.
        </p>
        <CodeBlock code="azd rest <method> <url> [flags]" language="bash" />
      </section>

      <section class="section">
        <h2>Commands</h2>
        <div class="command-grid">
${commandCards}
        </div>
      </section>

      <section class="section">
        <h2>Global Flags</h2>
        <p>These flags are available for all HTTP method commands:</p>

        <div class="flags-table">
          <table>
            <thead>
              <tr>
                <th>Flag</th>
                <th>Short</th>
                <th>Type</th>
                <th>Default</th>
                <th>Description</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><code>--scope</code></td>
                <td><code>-s</code></td>
                <td>string</td>
                <td>(auto-detect)</td>
                <td>OAuth scope for authentication. Auto-detected from URL if not provided.</td>
              </tr>
              <tr>
                <td><code>--no-auth</code></td>
                <td></td>
                <td>bool</td>
                <td>false</td>
                <td>Skip authentication (no bearer token). Automatically set for HTTP URLs.</td>
              </tr>
              <tr>
                <td><code>--header</code></td>
                <td><code>-H</code></td>
                <td>string[]</td>
                <td>[]</td>
                <td>Custom headers (repeatable, format: Key:Value).</td>
              </tr>
              <tr>
                <td><code>--data</code></td>
                <td><code>-d</code></td>
                <td>string</td>
                <td></td>
                <td>Request body (JSON string).</td>
              </tr>
              <tr>
                <td><code>--data-file</code></td>
                <td></td>
                <td>string</td>
                <td></td>
                <td>Read request body from file (also accepts @&#123;file&#125; shorthand).</td>
              </tr>
              <tr>
                <td><code>--output-file</code></td>
                <td></td>
                <td>string</td>
                <td></td>
                <td>Write response to file (raw for binary content).</td>
              </tr>
              <tr>
                <td><code>--format</code></td>
                <td><code>-f</code></td>
                <td>string</td>
                <td>auto</td>
                <td>Output format: <code>auto</code>, <code>json</code>, <code>raw</code>.</td>
              </tr>
              <tr>
                <td><code>--verbose</code></td>
                <td><code>-v</code></td>
                <td>bool</td>
                <td>false</td>
                <td>Verbose output (show headers, timing, redacted tokens).</td>
              </tr>
              <tr>
                <td><code>--paginate</code></td>
                <td></td>
                <td>bool</td>
                <td>false</td>
                <td>Follow continuation tokens/next links when supported.</td>
              </tr>
              <tr>
                <td><code>--retry</code></td>
                <td></td>
                <td>int</td>
                <td>3</td>
                <td>Retry attempts with exponential backoff for transient errors.</td>
              </tr>
              <tr>
                <td><code>--binary</code></td>
                <td></td>
                <td>bool</td>
                <td>false</td>
                <td>Stream request/response as binary without transformation.</td>
              </tr>
              <tr>
                <td><code>--insecure</code></td>
                <td><code>-k</code></td>
                <td>bool</td>
                <td>false</td>
                <td>Skip TLS certificate verification (use only for testing).</td>
              </tr>
              <tr>
                <td><code>--timeout</code></td>
                <td><code>-t</code></td>
                <td>duration</td>
                <td>30s</td>
                <td>Request timeout (e.g., 30s, 5m).</td>
              </tr>
              <tr>
                <td><code>--follow-redirects</code></td>
                <td></td>
                <td>bool</td>
                <td>true</td>
                <td>Follow HTTP redirects automatically.</td>
              </tr>
              <tr>
                <td><code>--max-redirects</code></td>
                <td></td>
                <td>int</td>
                <td>10</td>
                <td>Maximum redirect hops before stopping.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="section">
        <h2>Usage Examples</h2>

        <h3>Basic GET Request</h3>
        <CodeBlock code={\`azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01\`} language="bash" />

        <h3>POST with JSON Body</h3>
        <CodeBlock code={postExample} language="bash" />

        <h3>POST with File</h3>
        <CodeBlock code={\`azd rest post https://api.example.com/resource \\\\
  --data-file ./payload.json\`} language="bash" />

        <h3>Custom Headers</h3>
        <CodeBlock code={\`azd rest get https://api.example.com/resource \\\\
  --header "X-Custom-Header: value" \\\\
  --header "Accept: application/json"\`} language="bash" />

        <h3>Verbose Output</h3>
        <CodeBlock code={\`azd rest get https://management.azure.com/subscriptions?api-version=2020-01-01 \\\\
  --verbose\`} language="bash" />

        <h3>Save Response to File</h3>
        <CodeBlock code={\`azd rest get https://api.example.com/data \\\\
  --output-file response.json\`} language="bash" />

        <h3>Custom Scope</h3>
        <CodeBlock code={\`azd rest get https://api.myservice.com/data \\\\
  --scope https://myservice.com/.default\`} language="bash" />
      </section>

      <section class="section">
        <h2>Scope Detection</h2>
        <p>
          azd rest automatically detects the correct OAuth scope based on the URL. Supported services include:
        </p>
        <ul>
          <li>Management API (<code>management.azure.com</code>)</li>
          <li>Storage (<code>*.blob.core.windows.net</code>, <code>*.queue.core.windows.net</code>, etc.)</li>
          <li>Key Vault (<code>*.vault.azure.net</code>)</li>
          <li>Microsoft Graph (<code>graph.microsoft.com</code>)</li>
          <li>Azure DevOps (<code>dev.azure.com</code>, <code>*.visualstudio.com</code>)</li>
          <li>And 15+ more Azure services</li>
        </ul>
        <p>
          See <a href={\`\${base}reference/cli/\`}>Azure Scopes Reference</a> for the complete list.
        </p>
      </section>
    </div>
  </div>
</Layout>

<style>
  .page-container {
    max-width: 1280px;
    margin: 0 auto;
    padding: 2rem 1.5rem;
  }

  .page-header {
    margin-bottom: 3rem;
    text-align: center;
  }

  .page-header h1 {
    font-size: 2.5rem;
    font-weight: 700;
    margin: 0 0 1rem;
    color: var(--color-text-primary);
  }

  .page-intro {
    font-size: 1.125rem;
    color: var(--color-text-secondary);
  }

  .content {
    max-width: 1280px;
  }

  .section {
    margin-bottom: 3rem;
  }

  .section h2 {
    font-size: 1.75rem;
    font-weight: 600;
    margin: 0 0 1rem;
    color: var(--color-text-primary);
  }

  .section h3 {
    font-size: 1.25rem;
    font-weight: 600;
    margin: 1.5rem 0 1rem;
    color: var(--color-text-primary);
  }

  .section p {
    color: var(--color-text-secondary);
    line-height: 1.7;
    margin: 0 0 1rem;
  }

  .section ul {
    color: var(--color-text-secondary);
    line-height: 1.7;
    margin: 0 0 1rem;
    padding-left: 1.5rem;
  }

  .command-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 1.5rem;
    margin: 2rem 0;
  }

  .command-card {
    display: block;
    padding: 1.5rem;
    border: 1px solid var(--color-border-default);
    border-radius: 0.5rem;
    background-color: var(--color-bg-primary);
    text-decoration: none;
    transition: border-color 0.15s ease;
  }

  .command-card:hover {
    border-color: var(--color-interactive-default);
    text-decoration: none;
  }

  .command-card h3 {
    font-size: 1.125rem;
    font-weight: 600;
    margin: 0 0 0.5rem;
    color: var(--color-interactive-default);
  }

  .command-card p {
    color: var(--color-text-secondary);
    font-size: 0.9375rem;
    margin: 0 0 0.75rem;
  }

  .command-meta {
    font-size: 0.75rem;
    color: var(--color-text-secondary);
  }

  .flags-table {
    overflow-x: auto;
    margin: 1.5rem 0;
  }

  .flags-table table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.9375rem;
  }

  .flags-table th {
    background-color: var(--color-bg-secondary);
    padding: 0.75rem;
    text-align: left;
    font-weight: 600;
    color: var(--color-text-primary);
    border-bottom: 2px solid var(--color-border-default);
  }

  .flags-table td {
    padding: 0.75rem;
    border-bottom: 1px solid var(--color-border-default);
    color: var(--color-text-secondary);
  }

  .flags-table code {
    background-color: var(--color-bg-secondary);
    padding: 0.125rem 0.375rem;
    border-radius: 0.25rem;
    font-size: 0.875rem;
  }
</style>
`;
}

async function main() {
  console.log('🔧 Generating CLI reference pages...\n');

  // Discover commands
  const commands = discoverCommands();

  if (commands.length === 0) {
    console.log('  ⏭️  No commands discovered. Preserving existing pages.\n');
    return;
  }

  console.log(`  📋 Discovered ${commands.length} commands: ${commands.map(c => c.name).join(', ')}\n`);

  // Ensure output directory exists
  if (!fs.existsSync(OUTPUT_DIR)) {
    fs.mkdirSync(OUTPUT_DIR, { recursive: true });
  }

  // Generate index page
  const indexPage = generateIndexPage(commands);
  fs.writeFileSync(path.join(OUTPUT_DIR, 'index.astro'), indexPage);
  console.log(`  ✓ Generated: reference/cli/index.astro`);

  // Generate individual command pages
  for (const cmd of commands) {
    const page = generateCommandPage(cmd);
    fs.writeFileSync(path.join(OUTPUT_DIR, `${cmd.name}.astro`), page);
    console.log(`  ✓ Generated: reference/cli/${cmd.name}.astro`);
  }

  console.log(`\n✅ Generated ${commands.length + 1} CLI reference pages`);
}

main().catch(err => {
  console.error('Error generating CLI reference:', err);
  process.exit(1);
});
