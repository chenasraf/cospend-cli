# cospend-cli

**`cospend-cli`** is a command-line interface for managing expenses in
[Nextcloud Cospend](https://apps.nextcloud.com/apps/cospend) projects. It provides a quick way to
add and list expenses directly from your terminal without opening the web interface.

![Release](https://img.shields.io/github/v/release/chenasraf/cospend-cli)
![Downloads](https://img.shields.io/github/downloads/chenasraf/cospend-cli/total)
![License](https://img.shields.io/github/license/chenasraf/cospend-cli)

---

## Features

- **Add**, **list**, and **delete** expenses in Cospend projects via the **REST API**
- **List projects** you have access to
- **Filter** expenses by payer, owed members, amount, name, category, payment method, or date
- Resolve categories, payment methods, and members by **name or ID**
- **Case-insensitive** matching for all lookups
- **Currency code support** (e.g., `usd`, `eur`, `gbp`) with automatic symbol resolution
- **Local caching** of project data with 1-hour TTL for faster subsequent calls
- **Global project flag** - set `-p` before the command for easy shell aliases
- **Secure browser login** - OAuth-style authentication with 2FA support
- Cross-platform support: **macOS**, **Linux**, and **Windows**

---

## Installation

### Download Precompiled Binaries

Precompiled binaries for `cospend-cli` are available for **Linux**, **macOS**, and **Windows**:

- Visit the [Releases Page](https://github.com/chenasraf/cospend-cli/releases/latest) to download
  the latest version for your platform.

### Homebrew (macOS/Linux)

```bash
brew install chenasraf/tap/cospend-cli
```

### Build from Source

```bash
go install github.com/chenasraf/cospend-cli@latest
```

---

## Configuration

### Quick Setup (Recommended)

Run the interactive setup wizard:

```bash
cospend init
```

This will prompt for your Nextcloud domain and let you choose an authentication method using an
interactive selector (use arrow keys or j/k to navigate, Enter to select):

```
Choose login method:
  > Browser login (recommended) - Opens browser for secure authentication
    Password/App token - Enter credentials manually
```

- **Browser login (recommended)** - Opens your browser for secure OAuth-style authentication.
  Handles 2FA automatically and generates an app-specific password.

- **Password/App token** - Enter your credentials manually (useful for headless servers).

You can specify the config format with `--format`:

```bash
cospend init --format yaml
cospend init --format toml
cospend init --format json  # default
```

### Config File

The config file is searched in the following locations (in order of preference):

| OS      | Primary Location                                  | Fallback Location                            |
| ------- | ------------------------------------------------- | -------------------------------------------- |
| Linux   | `~/.config/cospend/cospend.{json,yaml,toml}`      | -                                            |
| macOS   | `~/Library/Application Support/cospend/cospend.*` | `~/.config/cospend/cospend.{json,yaml,toml}` |
| Windows | `%APPDATA%\cospend\cospend.*`                     | -                                            |

Example config files:

```json
{
  "domain": "https://cloud.example.com",
  "user": "alice",
  "password": "your-app-password"
}
```

```yaml
domain: https://cloud.example.com
user: alice
password: your-app-password
```

```toml
domain = "https://cloud.example.com"
user = "alice"
password = "your-app-password"
```

### Environment Variables

You can also use environment variables, which override config file values:

| Variable             | Description                          |
| -------------------- | ------------------------------------ |
| `NEXTCLOUD_DOMAIN`   | Your Nextcloud instance URL          |
| `NEXTCLOUD_USER`     | Your Nextcloud username              |
| `NEXTCLOUD_PASSWORD` | Your Nextcloud password or app token |

```bash
export NEXTCLOUD_DOMAIN="https://cloud.example.com"
export NEXTCLOUD_USER="alice"
export NEXTCLOUD_PASSWORD="your-app-password"
```

> **Tip:** For security, consider using a Nextcloud
> [app password](https://docs.nextcloud.com/server/latest/user_manual/en/session_management.html#managing-devices)
> instead of your main password.

---

## Usage

### Global Flags

The `-p`/`--project` flag can be used before or after the command, making it easy to create shell
aliases:

```bash
# These are equivalent:
cospend add "Groceries" 25.50 -p myproject
cospend -p myproject add "Groceries" 25.50

# Create an alias for a specific project:
alias cospend-home="cospend -p homeproject"
cospend-home add "Groceries" 25.50
cospend-home list
```

---

### Adding Expenses

```bash
cospend add <name> <amount> [flags]
```

#### Examples

```bash
# Add a simple expense
cospend add "Groceries" 25.50 -p myproject

# Add an expense with category and split between members
cospend add "Dinner" 45.00 -p myproject -c restaurant -f alice -f bob

# Add an expense paid by someone else
cospend add "Gas" 60.00 -p roadtrip -b charlie -f alice -f bob -f charlie

# Add an expense with payment method and comment
cospend add "Hotel" 150.00 -p vacation -m "credit card" -o "2 nights"

# Add an expense in a different currency
cospend add "Souvenirs" 30.00 -p vacation -C usd
```

#### Add Command Flags

| Short | Long         | Description                                               |
| ----- | ------------ | --------------------------------------------------------- |
| `-p`  | `--project`  | Project ID (required)                                     |
| `-c`  | `--category` | Category by ID or case-insensitive name                   |
| `-b`  | `--by`       | Paying member username (defaults to authenticated user)   |
| `-f`  | `--for`      | Owed member username (repeatable; defaults to payer only) |
| `-C`  | `--convert`  | Currency to convert to (by ID, name, or code like `usd`)  |
| `-m`  | `--method`   | Payment method by ID or case-insensitive name             |
| `-o`  | `--comment`  | Additional details about the bill                         |
| `-h`  | `--help`     | Display help information                                  |

---

### Listing Expenses

```bash
cospend list [flags]
cospend ls [flags]    # alias
```

#### Examples

```bash
# List all expenses in a project
cospend list -p myproject

# Filter by paying member
cospend list -p myproject -b alice

# Filter by category
cospend list -p myproject -c groceries

# Filter by amount (supports =, >, <, >=, <=)
cospend list -p myproject --amount ">50"
cospend list -p myproject --amount "<=100"

# Filter by name (case-insensitive, contains)
cospend list -p myproject -n dinner

# Filter by date (supports =, >, <, >=, <=)
cospend list -p myproject --date ">=2026-01-01"
cospend list -p myproject --date "<=01-15"        # short MM-DD format (assumes current year)

# Filter by current month or week
cospend list -p myproject --this-month
cospend list -p myproject --this-week

# Filter recent bills (d=days, w=weeks, m=months)
cospend list -p myproject --recent 7d
cospend list -p myproject --recent 2w
cospend list -p myproject --recent 1m

# Combine multiple filters
cospend list -p myproject -b alice -c restaurant --amount ">=20"
```

#### List Command Flags

| Short | Long           | Description                                                    |
| ----- | -------------- | -------------------------------------------------------------- |
| `-p`  | `--project`    | Project ID (required)                                          |
| `-b`  | `--by`         | Filter by paying member username                               |
| `-f`  | `--for`        | Filter by owed member username (repeatable)                    |
| `-a`  | `--amount`     | Filter by amount (e.g., `50`, `>30`, `<=100`, `=25`)           |
| `-n`  | `--name`       | Filter by name (case-insensitive, contains)                    |
| `-c`  | `--category`   | Filter by category name or ID                                  |
| `-m`  | `--method`     | Filter by payment method name or ID                            |
| `-l`  | `--limit`      | Limit number of results (0 = no limit)                         |
|       | `--date`       | Filter by date (e.g., `2026-01-15`, `>=2026-01-01`, `<=01-15`) |
|       | `--this-month` | Filter bills from the current month                            |
|       | `--this-week`  | Filter bills from the current calendar week                    |
|       | `--recent`     | Filter recent bills (e.g., `7d`, `2w`, `1m`)                   |
| `-h`  | `--help`       | Display help information                                       |

The output includes the bill ID for each expense, which can be used with the delete command.

---

### Deleting Expenses

```bash
cospend delete <bill_id> [flags]
cospend rm <bill_id> [flags]      # alias
```

#### Examples

```bash
# Delete a bill by ID (use 'cospend list' to find bill IDs)
cospend delete 123 -p myproject
```

#### Delete Command Flags

| Short | Long        | Description              |
| ----- | ----------- | ------------------------ |
| `-p`  | `--project` | Project ID (required)    |
| `-h`  | `--help`    | Display help information |

---

### Listing Projects

```bash
cospend projects [flags]
cospend proj [flags]    # alias
```

#### Examples

```bash
# List all active projects
cospend projects

# Include archived projects
cospend projects --all
```

#### Projects Command Flags

| Short | Long     | Description                          |
| ----- | -------- | ------------------------------------ |
| `-a`  | `--all`  | Show all projects including archived |
| `-h`  | `--help` | Display help information             |

---

## Caching

Project data (members, categories, payment methods, currencies) is cached locally to avoid repeated
API calls. The cache is stored in:

| OS      | Location                        |
| ------- | ------------------------------- |
| Linux   | `~/.cache/cospend-cli/`         |
| macOS   | `~/Library/Caches/cospend-cli/` |
| Windows | `%LOCALAPPDATA%\cospend-cli\`   |

Cache entries expire after **1 hour**. To force a refresh, simply delete the cache file for your
project.

---

## Currency Codes

When using the `-C` flag, you can specify currencies by:

1. **Numeric ID** - The Cospend currency ID
2. **Name** - The currency name as configured in Cospend (case-insensitive)
3. **Currency code** - Standard codes like `usd`, `eur`, `gbp`, `jpy`, etc.

Currency codes are automatically mapped to their symbols (e.g., `usd` -> `$`, `eur` -> `€`) and
matched against your project's configured currencies.

---

## Contributing

I am developing this package on my free time, so any support, whether code, issues, or just stars is
very helpful to sustaining its life. If you are feeling incredibly generous and would like to donate
just a small amount to help sustain this project, I would be very very thankful!

<a href='https://ko-fi.com/casraf' target='_blank'>
  <img height='36' style='border:0px;height:36px;'
    src='https://cdn.ko-fi.com/cdn/kofi1.png?v=3'
    alt='Buy Me a Coffee at ko-fi.com' />
</a>

I welcome any issues or pull requests on GitHub. If you find a bug, or would like a new feature,
don't hesitate to open an appropriate issue and I will do my best to reply promptly.

---

## License

`cospend-cli` is licensed under the [MIT License](/LICENSE).
