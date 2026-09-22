# Installs the workshop prerequisites on Windows.
#
#   .\scripts\install-prereqs.ps1          # install what is missing
#   .\scripts\install-prereqs.ps1 -Check   # report only, install nothing
#
# PowerShell rather than Node, because Node is one of the things it installs.
# Run from PowerShell, not CMD. Administrator is not required.

[CmdletBinding()]
param(
	[switch]$Check
)

$ErrorActionPreference = 'Continue'

$NodeMin = 24
$GoMin = [version]'1.25'
$PnpmMajor = 10

$script:Failed = $false
$script:Summary = @()

function Write-Step($text) { Write-Host "`n==> $text" -ForegroundColor White }
function Write-Info($text) { Write-Host "    $text" }
function Write-Warn($text) { Write-Host "    $text" -ForegroundColor Yellow }
function Write-Fail($text) { Write-Host "    $text" -ForegroundColor Red; $script:Failed = $true }

function Test-Have($name) {
	$null -ne (Get-Command $name -ErrorAction SilentlyContinue)
}

function Add-Row($name, $status, $detail) {
	$script:Summary += [pscustomobject]@{ Tool = $name; Status = $status; Detail = $detail }
}

# Installs the first winget id that resolves, or reports them when -Check is set.
# Package ids drift between winget releases, so each caller passes fallbacks.
function Invoke-Winget([string[]]$ids) {
	if ($Check) {
		Write-Info "would run: winget install --id $($ids[0])"
		return $false
	}
	foreach ($id in $ids) {
		winget install --id $id --accept-source-agreements --accept-package-agreements --silent
		if ($LASTEXITCODE -eq 0) { return $true }
		Write-Warn "winget could not install $id"
	}
	Write-Info "search for it yourself with: winget search $($ids[0].Split('.')[0])"
	return $false
}

# Re-reads the machine and user PATH so a just-installed tool is callable here.
function Update-Path {
	$machine = [Environment]::GetEnvironmentVariable('Path', 'Machine')
	$user = [Environment]::GetEnvironmentVariable('Path', 'User')
	$env:Path = "$machine;$user"
}

# ---------------------------------------------------------------- winget

function Test-Winget {
	Write-Step 'winget'
	if (Test-Have winget) {
		Write-Info (winget --version)
		return $true
	}
	Write-Fail 'winget is missing. Update App Installer from the Microsoft Store, then rerun.'
	Write-Info 'Alternatively install each tool by hand; see SETUP.md.'
	Add-Row 'winget' 'missing' 'Microsoft Store > App Installer'
	return $false
}

# ---------------------------------------------------------------- node

function Install-Node {
	Write-Step "Node $NodeMin+"
	if (Test-Have node) {
		$v = (node --version).TrimStart('v')
		if ([int]($v.Split('.')[0]) -ge $NodeMin) {
			Write-Info "node $v"
			Add-Row 'Node' 'ok' $v
			return
		}
		Write-Warn "node $v is older than $NodeMin"
	} else {
		Write-Info 'not installed'
	}

	# The LTS package tracks 24, which satisfies the minimum.
	Invoke-Winget @('OpenJS.NodeJS.LTS', 'OpenJS.NodeJS') | Out-Null
	Update-Path

	if ((Test-Have node) -and [int]((node --version).TrimStart('v').Split('.')[0]) -ge $NodeMin) {
		Add-Row 'Node' 'installed' (node --version).TrimStart('v')
	} else {
		Add-Row 'Node' 'missing' 'https://nodejs.org'
		if (-not $Check) { Write-Fail 'node is still missing or too old' }
	}
}

# ---------------------------------------------------------------- pnpm

function Install-Pnpm {
	Write-Step "pnpm $PnpmMajor"
	if (Test-Have pnpm) {
		$v = pnpm --version
		if ($v.Split('.')[0] -eq "$PnpmMajor") {
			Write-Info "pnpm $v"
			Add-Row 'pnpm' 'ok' $v
			return
		}
		Write-Warn "pnpm $v is not $PnpmMajor.x, and this repo's lockfile was written for $PnpmMajor"
	} else {
		Write-Info 'not installed'
	}

	if (-not (Test-Have npm)) {
		Write-Fail 'npm is missing, so Node did not install. Fix Node first.'
		Add-Row 'pnpm' 'missing' 'needs Node'
		return
	}

	# Pinned: an unpinned install pulls pnpm 12, which reads the workspace
	# settings block differently.
	if ($Check) {
		Write-Info "would run: npm install -g pnpm@$PnpmMajor"
	} else {
		npm install -g "pnpm@$PnpmMajor"
		Update-Path
	}

	if ((Test-Have pnpm) -and (pnpm --version).Split('.')[0] -eq "$PnpmMajor") {
		Add-Row 'pnpm' 'installed' (pnpm --version)
	} else {
		Add-Row 'pnpm' 'missing' "npm install -g pnpm@$PnpmMajor"
		if (-not $Check) { Write-Fail 'pnpm is still missing' }
	}
}

# ---------------------------------------------------------------- go

function Install-Go {
	Write-Step "Go $GoMin+"
	if (Test-Have go) {
		$v = ((go version) -split ' ')[2].TrimStart('go')
		if ([version]($v -replace '(\d+\.\d+).*', '$1') -ge $GoMin) {
			Write-Info "go $v"
			Add-Row 'Go' 'ok' $v
			return
		}
		Write-Warn "go $v is older than $GoMin"
	} else {
		Write-Info 'not installed'
	}

	Invoke-Winget @('GoLang.Go') | Out-Null
	Update-Path

	if (Test-Have go) {
		Add-Row 'Go' 'installed' ((go version) -split ' ')[2].TrimStart('go')
	} else {
		Add-Row 'Go' 'missing' 'https://go.dev/dl/'
		if (-not $Check) { Write-Fail 'go is still missing' }
	}
}

# ---------------------------------------------------------------- podman

function Install-Podman {
	Write-Step 'Podman, with a compose provider'
	if (Test-Have podman) {
		Write-Info (podman --version)
	} else {
		Write-Info 'not installed'
		# Podman Desktop carries the CLI and installs a compose provider.
		Invoke-Winget @('RedHat.Podman-Desktop', 'RedHat.Podman') | Out-Null
		Update-Path
	}

	if (Test-Have podman) {
		podman compose version 2>$null | Out-Null
		if ($LASTEXITCODE -ne 0) {
			Write-Warn 'no compose provider yet. Open Podman Desktop and accept its Compose install.'
		}
		Add-Row 'Podman' 'ok' ((podman --version) -split ' ')[2]
	} else {
		Add-Row 'Podman' 'missing' 'https://podman-desktop.io'
		if (-not $Check) { Write-Fail 'podman is still missing' }
	}

	Write-Info 'Open Podman Desktop once and start the machine it offers before pnpm start.'
}

# ---------------------------------------------------------------- python

function Install-Python {
	Write-Step 'Python 3.9+'
	if (Test-Have python3) {
		Write-Info (python3 --version)
		Add-Row 'Python' 'ok' ((python3 --version) -split ' ')[1]
		return
	}
	if (Test-Have python) {
		Write-Info (python --version)
		Add-Row 'Python' 'ok' ((python --version) -split ' ')[1]
		return
	}
	Write-Info 'not installed'
	Invoke-Winget @('Python.Python.3.13', 'Python.Python.3.12', 'Python.Python.3.11') | Out-Null
	Update-Path

	if ((Test-Have python) -or (Test-Have python3)) {
		Add-Row 'Python' 'installed' 'on PATH'
	} else {
		Add-Row 'Python' 'missing' 'https://www.python.org'
		Write-Warn 'If you install by hand, tick "Add python.exe to PATH" on the first screen.'
	}
}

# ---------------------------------------------------------------- claude code

function Install-Claude {
	Write-Step 'Claude Code'
	if (Test-Have claude) {
		Write-Info (claude --version)
		Add-Row 'Claude Code' 'ok' ((claude --version) -split ' ')[0]
		return
	}
	Write-Info 'not installed'
	if ($Check) {
		Write-Info 'would run: irm https://claude.ai/install.ps1 | iex'
	} else {
		Invoke-RestMethod https://claude.ai/install.ps1 | Invoke-Expression
		Update-Path
	}

	if (Test-Have claude) {
		Add-Row 'Claude Code' 'installed' ((claude --version) -split ' ')[0]
	} else {
		Add-Row 'Claude Code' 'missing' 'open a new terminal, then run claude --version'
	}
}

# ---------------------------------------------------------------- summary

function Write-Summary {
	Write-Host "`n==> Summary`n" -ForegroundColor White
	foreach ($row in $script:Summary) {
		$colour = if ($row.Status -in @('ok', 'installed')) { 'Green' } else { 'Red' }
		Write-Host ("    {0,-14}" -f $row.Tool) -NoNewline
		Write-Host ("{0,-11}" -f $row.Status) -ForegroundColor $colour -NoNewline
		Write-Host " $($row.Detail)"
	}

	if ($script:Failed) {
		Write-Host "`n    Something is still missing. Fix the red rows above, then rerun." -ForegroundColor Red
	} elseif ($Check) {
		Write-Host "`n    Check only. Drop -Check to install anything marked missing."
	} else {
		Write-Host @'

    Open a new terminal so every PATH change takes effect, then:

        cd path\to\ai-day-demo
        pnpm start

    Then open http://localhost:5173
'@
	}
}

# ---------------------------------------------------------------- main

$arch = $env:PROCESSOR_ARCHITECTURE
Write-Host "Workshop prerequisites  (Windows $arch, PowerShell $($PSVersionTable.PSVersion))" -ForegroundColor White
if ($Check) { Write-Info 'check only, nothing will be installed' }

if (Test-Winget) {
	Install-Node
	Install-Pnpm
	Install-Go
	Install-Podman
	Install-Python
	Install-Claude
}
Write-Summary

if ($script:Failed) { exit 1 } else { exit 0 }
