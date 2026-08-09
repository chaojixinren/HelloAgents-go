# Security Policy

## Supported Versions

HelloAgents-Go is under active development. Security fixes are applied to the
latest released version and the `main` branch only.

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

The HelloAgents-Go maintainers take security bugs seriously. We appreciate
your efforts to responsibly disclose your findings, and will make every effort
to acknowledge your contributions.

### How to report

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please report vulnerabilities **privately** by emailing the
maintainer at **chaoji_xinren@163.com** with the subject line starting with
`[SECURITY] HelloAgents-Go`.

Please include the following in your report:

- A description of the vulnerability and its impact
- The affected version(s) / commit SHA
- Step-by-step reproduction instructions or a proof-of-concept
- Any suggested mitigations (optional)

### Response timeline

| Stage                      | Target SLA |
| -------------------------- | ---------- |
| Acknowledgment of receipt  | 48 hours   |
| Initial assessment         | 5 business days |
| Fix or mitigation released | 30 days (severity dependent) |

You will be kept informed of progress at each stage. Once a fix is released
and verified, we will publish a public advisory crediting the reporter
(unless they prefer to remain anonymous).

### Scope

The following are considered in-scope for security reports:

- The framework code under `hello_agents/` and `cmd/`
- Built-in tools (e.g. file read/write, command execution, HTTP)
- LLM adapter credential handling and transport
- CI/release workflow configurations under `.github/`

### Out of scope

- Vulnerabilities in third-party dependencies (report upstream)
- Issues requiring already-compromised API credentials
- Social engineering of maintainers

## Security considerations when using this framework

HelloAgents-Go lets agents run tools that can read/write files and (when the
`BashTool` is registered) execute arbitrary shell commands. Before running an
agent against untrusted inputs:

- Register only the tools the agent actually needs.
- Run the agent with the minimum filesystem permissions necessary.
- Treat any LLM-generated tool call as untrusted input.
- Keep your `LLM_API_KEY` and `.env` file out of version control (they are
  covered by [.gitignore](.gitignore)).
