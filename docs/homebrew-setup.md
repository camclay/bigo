# Homebrew Distribution

This guide covers setting up Homebrew distribution for BigO.

## Option 1: Personal Tap (Recommended)

A "tap" is a custom Homebrew repository. Users install with:

```bash
brew tap yourusername/tap
brew install bigo
```

### Step 1: Create the Tap Repository

Create a new GitHub repo named `homebrew-tap` (the `homebrew-` prefix is required):

```bash
gh repo create homebrew-tap --public --description "Homebrew formulas"
cd homebrew-tap
mkdir Formula
```

### Step 2: Create the Formula

Create `Formula/bigo.rb`:

```ruby
class Bigo < Formula
  desc "Unified Claude + Ollama Agent Orchestrator"
  homepage "https://github.com/yourusername/bigo"
  version "1.0.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/yourusername/bigo/releases/download/v#{version}/bigo-darwin-arm64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    else
      url "https://github.com/yourusername/bigo/releases/download/v#{version}/bigo-darwin-amd64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/yourusername/bigo/releases/download/v#{version}/bigo-linux-arm64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    else
      url "https://github.com/yourusername/bigo/releases/download/v#{version}/bigo-linux-amd64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
  end

  def install
    binary_name = "bigo-#{OS.kernel_name.downcase}-#{Hardware::CPU.arch}"
    bin.install binary_name => "bigo"
  end

  test do
    assert_match "BigO", shell_output("#{bin}/bigo --version")
  end
end
```

### Step 3: Commit and Push

```bash
git add Formula/bigo.rb
git commit -m "Add bigo formula"
git push
```

### Step 4: Test Installation

```bash
brew tap yourusername/tap
brew install bigo
bigo --version
```

## Option 2: Build from Source Formula

If you prefer building from source (more portable but slower install):

```ruby
class Bigo < Formula
  desc "Unified Claude + Ollama Agent Orchestrator"
  homepage "https://github.com/yourusername/bigo"
  url "https://github.com/yourusername/bigo/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "REPLACE_WITH_TARBALL_SHA256"
  license "MIT"
  head "https://github.com/yourusername/bigo.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X main.version=#{version}
    ]
    system "go", "build", *std_go_args(ldflags:), "./cmd/bigo"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/bigo --version")

    # Test init command
    system "#{bin}/bigo", "init"
    assert_predicate testpath/".bigo/ledger.db", :exist?
  end
end
```

## Automatic Formula Updates

Add this workflow to your main bigo repo to auto-update the tap on release:

See `.github/workflows/homebrew.yaml` in this repository.

## Option 3: Homebrew Core (Future)

Once BigO has significant adoption, you can submit to Homebrew Core:

**Requirements:**
- 50+ GitHub stars (soft requirement)
- Stable releases
- Active maintenance
- Passes `brew audit --strict`

**Process:**
1. Fork https://github.com/Homebrew/homebrew-core
2. Add formula to `Formula/b/bigo.rb`
3. Run `brew audit --strict --new bigo`
4. Submit PR

## Troubleshooting

### "SHA256 mismatch"

Regenerate checksums:
```bash
shasum -a 256 bigo-darwin-arm64
```

### "No bottle available"

Bottles are pre-built binaries. Personal taps don't have bottles by default.
Users will build from source or download the binary.

### Testing the formula locally

```bash
brew install --build-from-source ./Formula/bigo.rb
brew audit --strict ./Formula/bigo.rb
brew test ./Formula/bigo.rb
```
