# Installs the workshop prerequisites from the offline bundle, on Windows.
#
#   .\install-prereqs-offline.ps1              # install what is missing
#   .\install-prereqs-offline.ps1 -Check       # report only, install nothing
#   .\install-prereqs-offline.ps1 -Bundle DIR  # the bundle is somewhere else
#
# Needs no network, and no Administrator: everything lands under your profile.
# PowerShell rather than Node, because Node is one of the things it installs.

[CmdletBinding()]
param(
	[switch]$Check,
	[string]$Bundle
)

$ErrorActionPreference = 'Continue'

$NodeMin = 24
$GoMin = [version]'1.25'
$PnpmMajor = 10
$Platform = 'win32-x64'

# Per-user install root, so nothing here needs an elevated prompt.
$InstallRoot = Join-Path $env:LOCALAPPDATA 'ai-day-demo'

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

# ---------------------------------------------------------------- bundle

function Find-Bundle {
	if ($Bundle) { return (Resolve-Path $Bundle -ErrorAction SilentlyContinue).Path }
	$here = Split-Path -Parent $PSCommandPath
	$candidates = @(
		$here,
		(Join-Path $here 'prereq-bundle'),
		(Join-Path (Get-Location) 'prereq-bundle')
	)
	$parent = Split-Path -Parent $here
	if ($parent) { $candidates += (Join-Path $parent 'prereq-bundle') }
	foreach ($candidate in $candidates) {
		if ((Test-Path (Join-Path $candidate $Platform)) -and (Test-Path (Join-Path $candidate 'common'))) {
			return (Resolve-Path $candidate).Path
		}
	}
	return $null
}

$BundleDir = Find-Bundle
if (-not $BundleDir) {
	Write-Error 'no prereq-bundle found. Pass -Bundle C:\path\to\prereq-bundle'
	exit 2
}
$Assets = Join-Path $BundleDir $Platform
$Common = Join-Path $BundleDir 'common'
if (-not (Test-Path $Assets)) {
	Write-Error "$BundleDir has no $Platform directory. This bundle was built for other machines."
	exit 2
}

# Returns the first file in $dir matching $pattern, or $null.
function Find-Asset($dir, $pattern) {
	Get-ChildItem -Path $dir -Filter $pattern -File -ErrorAction SilentlyContinue |
		Select-Object -First 1 -ExpandProperty FullName
}

# Compares a bundled file against the SHA256SUMS beside it.
function Test-Asset($path) {
	$dir = Split-Path -Parent $path
	$name = Split-Path -Leaf $path
	$sums = Join-Path $dir 'SHA256SUMS'
	if (-not (Test-Path $sums)) {
		Write-Fail "no SHA256SUMS in $dir"
		return $false
	}
	$expected = $null
	foreach ($line in Get-Content $sums) {
		$parts = $line -split '\s+', 2
		if ($parts.Count -eq 2 -and $parts[1].Trim() -eq $name) { $expected = $parts[0] }
	}
	if (-not $expected) {
		Write-Fail "$name is not listed in SHA256SUMS"
		return $false
	}
	$actual = (Get-FileHash -Algorithm SHA256 -Path $path).Hash.ToLower()
	if ($actual -ne $expected.ToLower()) {
		Write-Fail "$name is corrupt. Copy the bundle across again."
		return $false
	}
	Write-Info "verified $name"
	return $true
}

# Adds a directory to the user PATH, and to this session.
function Add-UserPath($dir) {
	$current = [Environment]::GetEnvironmentVariable('Path', 'User')
	if ($current -split ';' -notcontains $dir) {
		[Environment]::SetEnvironmentVariable('Path', "$dir;$current", 'User')
		Write-Info "added $dir to your PATH"
	}
	$env:Path = "$dir;$env:Path"
}

# ---------------------------------------------------------------- node

function Install-Node {
	Write-Step "Node $NodeMin+"
	if (Test-Have node) {
		$current = (node --version).TrimStart('v')
		if ([int]($current.Split('.')[0]) -ge $NodeMin) {
			Write-Info "node $current"
			Add-Row 'Node' 'ok' $current
			return
		}
		Write-Warn "node $current is older than $NodeMin"
	} else {
		Write-Info 'not installed'
	}

	$zip = Find-Asset $Assets 'node-v*-win-x64.zip'
	if (-not $zip) {
		Write-Fail "no node zip in $Assets"
		Add-Row 'Node' 'missing' 'not in the bundle'
		return
	}
	if (-not (Test-Asset $zip)) { Add-Row 'Node' 'missing' 'corrupt download'; return }

	$target = Join-Path $InstallRoot 'node'
	if ($Check) {
		Write-Info "would expand $(Split-Path -Leaf $zip) into $target"
	} else {
		# The zip's top level is node-vX-win-x64\, which becomes the install dir itself.
		$staging = Join-Path $env:TEMP "node-unzip-$PID"
		Remove-Item -Recurse -Force $staging, $target -ErrorAction SilentlyContinue
		Expand-Archive -Path $zip -DestinationPath $staging -Force
		$inner = Get-ChildItem -Path $staging -Directory | Select-Object -First 1
		New-Item -ItemType Directory -Force -Path (Split-Path -Parent $target) | Out-Null
		Move-Item -Path $inner.FullName -Destination $target
		Remove-Item -Recurse -Force $staging -ErrorAction SilentlyContinue
		Add-UserPath $target
	}

	if ((Test-Have node) -and [int]((node --version).TrimStart('v').Split('.')[0]) -ge $NodeMin) {
		Add-Row 'Node' 'installed' (node --version).TrimStart('v')
	} else {
		Add-Row 'Node' 'missing' (Split-Path -Leaf $zip)
		if (-not $Check) { Write-Fail 'node is still missing or too old' }
	}
}

# ---------------------------------------------------------------- pnpm

function Install-Pnpm {
	Write-Step "pnpm $PnpmMajor"
	if (Test-Have pnpm) {
		$current = pnpm --version
		if ($current.Split('.')[0] -eq "$PnpmMajor") {
			Write-Info "pnpm $current"
			Add-Row 'pnpm' 'ok' $current
			return
		}
		Write-Warn "pnpm $current is not $PnpmMajor.x, and this repo's lockfile was written for $PnpmMajor"
	} else {
		Write-Info 'not installed'
	}

	if (-not (Test-Have npm)) {
		Write-Fail 'npm is missing, so Node did not install. Fix Node first.'
		Add-Row 'pnpm' 'missing' 'needs Node'
		return
	}

	$tgz = Find-Asset $Common 'pnpm-*.tgz'
	if (-not $tgz) {
		Write-Fail "no pnpm tarball in $Common"
		Add-Row 'pnpm' 'missing' 'not in the bundle'
		return
	}
	if (-not (Test-Asset $tgz)) { Add-Row 'pnpm' 'missing' 'corrupt download'; return }

	if ($Check) {
		Write-Info "would run: npm install -g $(Split-Path -Leaf $tgz)"
	} else {
		# The package bundles its own dependencies, so --offline resolves nothing.
		npm install -g --offline --no-audit --no-fund $tgz
	}

	if ((Test-Have pnpm) -and (pnpm --version).Split('.')[0] -eq "$PnpmMajor") {
		Add-Row 'pnpm' 'installed' (pnpm --version)
	} else {
		Add-Row 'pnpm' 'missing' (Split-Path -Leaf $tgz)
		if (-not $Check) { Write-Fail 'pnpm is still missing' }
	}
}

# ---------------------------------------------------------------- go

function Install-Go {
	Write-Step "Go $GoMin+"
	if (Test-Have go) {
		$current = ((go version) -split ' ')[2].TrimStart('go')
		if ([version]($current -replace '(\d+\.\d+).*', '$1') -ge $GoMin) {
			Write-Info "go $current"
			Add-Row 'Go' 'ok' $current
			return
		}
		Write-Warn "go $current is older than $GoMin"
	} else {
		Write-Info 'not installed'
	}

	$zip = Find-Asset $Assets 'go*.windows-amd64.zip'
	if (-not $zip) {
		Write-Fail "no go zip in $Assets"
		Add-Row 'Go' 'missing' 'not in the bundle'
		return
	}
	if (-not (Test-Asset $zip)) { Add-Row 'Go' 'missing' 'corrupt download'; return }

	$target = Join-Path $InstallRoot 'go'
	if ($Check) {
		Write-Info "would expand $(Split-Path -Leaf $zip) into $target"
	} else {
		# go.dev ships a whole tree under go\, and refuses to run when merged into an old one.
		Remove-Item -Recurse -Force $target -ErrorAction SilentlyContinue
		New-Item -ItemType Directory -Force -Path $InstallRoot | Out-Null
		Expand-Archive -Path $zip -DestinationPath $InstallRoot -Force
		Add-UserPath (Join-Path $target 'bin')
	}

	if (Test-Have go) {
		Add-Row 'Go' 'installed' ((go version) -split ' ')[2].TrimStart('go')
	} else {
		Add-Row 'Go' 'missing' (Split-Path -Leaf $zip)
		if (-not $Check) { Write-Fail 'go is still missing' }
	}
}

# ---------------------------------------------------------------- python

function Install-Python {
	Write-Step 'Python 3.9+'
	foreach ($name in @('python3', 'python')) {
		if (Test-Have $name) {
			Write-Info (& $name --version)
			Add-Row 'Python' 'ok' ((& $name --version) -split ' ')[1]
			return
		}
	}
	Write-Info 'not installed'

	$exe = Find-Asset $Assets 'python-*-amd64.exe'
	if (-not $exe) {
		Write-Fail "no python installer in $Assets"
		Add-Row 'Python' 'missing' 'not in the bundle'
		return
	}
	if (-not (Test-Asset $exe)) { Add-Row 'Python' 'missing' 'corrupt download'; return }

	if ($Check) {
		Write-Info "would run: $(Split-Path -Leaf $exe) /quiet InstallAllUsers=0 PrependPath=1"
	} else {
		# InstallAllUsers=0 keeps it per-user, which is what avoids the UAC prompt.
		Start-Process -FilePath $exe -Wait -ArgumentList @(
			'/quiet', 'InstallAllUsers=0', 'PrependPath=1', 'Include_test=0'
		)
		$machine = [Environment]::GetEnvironmentVariable('Path', 'Machine')
		$user = [Environment]::GetEnvironmentVariable('Path', 'User')
		$env:Path = "$machine;$user"
	}

	if ((Test-Have python) -or (Test-Have python3)) {
		Add-Row 'Python' 'installed' 'on PATH'
	} else {
		Add-Row 'Python' 'missing' (Split-Path -Leaf $exe)
		if (-not $Check) { Write-Fail 'python is still missing' }
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

	$binary = Find-Asset $Assets 'claude.exe'
	if (-not $binary) {
		Write-Fail "no claude.exe in $Assets"
		Add-Row 'Claude Code' 'missing' 'not in the bundle'
		return
	}
	if (-not (Test-Asset $binary)) { Add-Row 'Claude Code' 'missing' 'corrupt download'; return }

	$target = Join-Path $env:USERPROFILE '.local\bin'
	if ($Check) {
		Write-Info "would copy claude.exe into $target"
	} else {
		# `claude install` reads the release list over the network, so the offline
		# path is the binary on its own. It updates itself once there is internet.
		New-Item -ItemType Directory -Force -Path $target | Out-Null
		Copy-Item -Path $binary -Destination (Join-Path $target 'claude.exe') -Force
		Add-UserPath $target
	}

	if ($Check) {
		Add-Row 'Claude Code' 'missing' (Split-Path -Leaf $binary)
		return
	}
	if (Test-Have claude) {
		Add-Row 'Claude Code' 'installed' ((claude --version) -split ' ')[0]
	} else {
		Add-Row 'Claude Code' 'missing' 'open a new terminal, then run claude --version'
	}
}

# ---------------------------------------------------------------- not bundled

function Test-Podman {
	Write-Step 'Podman, with a compose provider'
	if (-not (Test-Have podman)) {
		Write-Fail 'not installed, and Podman is not in this bundle'
		Add-Row 'Podman' 'missing' 'https://podman-desktop.io'
		return
	}
	Write-Info (podman --version)
	podman compose version 2>$null | Out-Null
	if ($LASTEXITCODE -eq 0) {
		Add-Row 'Podman' 'ok' ((podman --version) -split ' ')[2]
	} else {
		Write-Warn 'no compose provider. Open Podman Desktop and accept its Compose install.'
		Add-Row 'Podman' 'missing' 'podman compose version reports no provider'
	}
	Write-Info 'Open Podman Desktop once and start the machine it offers before pnpm start.'
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

	if ($Check) {
		Write-Host "`n    Check only. Drop -Check to install anything marked missing."
	} elseif ($script:Failed) {
		Write-Host "`n    Something is still missing. Fix the red rows above, then rerun." -ForegroundColor Red
	} else {
		Write-Host @'

    Open a new terminal so every PATH change takes effect, then:

        cd path\to\ai-day-demo
        pnpm start
'@
	}
}

# ---------------------------------------------------------------- main

Write-Host "Workshop prerequisites, offline  (Windows $env:PROCESSOR_ARCHITECTURE, PowerShell $($PSVersionTable.PSVersion))" -ForegroundColor White
Write-Info "bundle $BundleDir"
Write-Info "prefix $InstallRoot"
if ($Check) { Write-Info 'check only, nothing will be installed' }

Install-Node
Install-Pnpm
Install-Go
Install-Python
Install-Claude
Test-Podman
Write-Summary

if ($script:Failed) { exit 1 } else { exit 0 }
