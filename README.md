# Termite

AI-powered security scanner for your code. Termite analyzes your source files for vulnerabilities using a locally-hosted language model, keeping your code private and off third-party servers.

---

## Features

- Static security analysis powered by qwen2.5-coder
- Scans Python, Go, JavaScript, and more
- Fast, lightweight CLI — single binary, no runtime dependencies
- Code never leaves your infrastructure
- Simple output with clear vulnerability descriptions

---

## Installation

### Option 1 — curl install script (recommended)

```bash
curl -sSL https://termite-sec.com/install.sh | bash
```

The script automatically detects your OS and architecture and installs the correct binary.

### Option 2 — Download binary manually

Download the latest binary for your platform from the [releases page](https://github.com/termite-sec/termite/releases):

| Platform | Binary |
|----------|--------|
| Linux x86_64 | `termite-linux-amd64` |
| macOS Intel | `termite-darwin-amd64` |
| macOS Apple Silicon | `termite-darwin-arm64` |
| Windows x86_64 | `termite-windows-amd64.exe` |

Then make it executable and move it to your PATH:

```bash
chmod +x termite-linux-amd64
sudo mv termite-linux-amd64 /usr/local/bin/termite
```

### Option 3 — Build from source

Requirements: Go 1.21 or higher

```bash
git clone https://github.com/termite-sec/termite.git
cd termite/cmd/dig
go build -o termite .
sudo mv termite /usr/local/bin/termite
```

---

## Configuration

On first run, termite will prompt you to configure the server endpoint. You can also configure it manually:

```bash
termite configure
```

By default, termite points to the hosted server at `api.termite-sec.com`. If you are self-hosting, set your own endpoint:

```bash
termite configure --server http://your-server:8080
```

---

## Usage

### Scan a single file

```bash
termite scan main.py
```

### Scan a directory

```bash
termite scan ./src
```

### Scan with verbose output

```bash
termite scan main.py --verbose
```

### Example output

```
Scanning: main.py
  found 1 file
  scanning with termite AI...

  [HIGH] SQL Injection — line 42
  User input is passed directly into a raw SQL query without sanitization.
  Recommendation: Use parameterized queries or a query builder.

  [MEDIUM] Hardcoded credential — line 17
  A string resembling an API key is hardcoded in the source.
  Recommendation: Move secrets to environment variables or a secrets manager.

  2 issues found.
```

---

## Self-Hosting

Termite's backend requires Ollama with qwen2.5-coder running locally or on a server you control.

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull the model
ollama pull qwen2.5-coder:7b

# Start Ollama
ollama serve
```

Then point termite at your server:

```bash
termite configure --server http://localhost:11434
```

---

## Contributing

Contributions are welcome. Please follow the steps below:

1. Fork the repository
2. Create a feature branch

```bash
git checkout -b feature/your-feature-name
```

3. Make your changes and commit

```bash
git commit -m "add: description of your change"
```

4. Push and open a pull request

```bash
git push origin feature/your-feature-name
```

### Guidelines

- Keep pull requests focused on a single change
- Write clear commit messages
- Test your changes before submitting
- Open an issue first for large changes or new features

---

## License

MIT License. See [LICENSE](LICENSE) for details.

---

## Contact

- Website: [termite-sec.com](https://termite-sec.com)
- Issues: [github.com/termite-sec/termite/issues](https://github.com/termite-sec/termite/issues)
