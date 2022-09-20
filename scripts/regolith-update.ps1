# Convert string to object with semver parts
function Convert-Semver {
    param (
        [string]$VersionString
    )
    $split = $VersionString.split('.')
    [pscustomobject]@{
        Major = [int]($split[0])
        Minor = [int]($split[1])
        Patch = [int]($split[2])
    }
}

# Whether the first parameter is greater than the second
function Test-Semver {
    param (
        [pscustomobject]$First,
        [pscustomobject]$Second
    )
    if ($First.Major -gt $Second.Major) {
        return $true
    }
    if ($First.Major -eq $Second.Major) {
        if ($First.Minor -gt $Second.Minor) {
            return $true
        }
        if ($First.Minor -eq $Second.Minor) {
            if ($First.Patch -gt $Second.Patch) {
                return $true
            }
        }
    }
    return $false
}
# Get the latest release from GitHub
try {
    $latestResponse = (Invoke-RestMethod -Uri https://api.github.com/repos/Bedrock-OSS/regolith/releases/latest)
} catch {
    # Check if the error is because we are rate limited
    if ($_.Exception.Response.StatusDescription -match 'rate limit exceeded') {
        Write-Host 'Too many requests to GitHub API. Try again later.'
        Write-Host Reset should happen: $((Get-Date -Date "1970-01-01 00:00:00Z").toUniversalTime().addSeconds($_.Exception.Response.Headers['X-RateLimit-Reset']))
    } else {
        # Otherwise, rethrow the error
        Write-Host 'Error contacting GitHub API. Try again later.'
        throw $_.Exception
    }
    exit 1
}

# Get the latest release Semver object
$latest = Convert-Semver $latestResponse.tag_name
# Get the current version Semver object
$current = Convert-Semver (regolith --version).split([Environment]::NewLine)[0].split(' ')[2]

# Compare versions
$result = Test-Semver $latest $current
if (!$result) {
    Write-Host "You are running the latest version of Regolith." -ForegroundColor Green
} else {
    # Inform about update and show changelog
    Write-Host "You are not running the latest version of Regolith." -ForegroundColor Yellow
    Write-Host "The latest version is $($latest.Major).$($latest.Minor).$($latest.Patch). The current version is $($current.Major).$($current.Minor).$($current.Patch)." -ForegroundColor Yellow
    Write-Host $([Environment]::NewLine)
    Write-Host $latestResponse.body -BackgroundColor DarkCyan
    Write-Host $([Environment]::NewLine) $([Environment]::NewLine)

    # Ask if the user wants to update
    $confirmation = Read-Host "Do you want to update? (y/n) "
    if ($confirmation -eq 'y' -or $confirmation -eq 'Y') {
        # Search for the MSI installer
        $asset = $null
        foreach ($a in $latestResponse.assets) {
            if ($a.name -match '.msi') {
                $asset = $a.browser_download_url
                break
            }
        }
        # Add progress activity with nice name
        Write-Progress -Activity "Downloading Installer"
        # Download installer
        $TempFile = New-TemporaryFile | Rename-Item -NewName { $_ -replace 'tmp$', 'msi' } -PassThru
        Invoke-WebRequest -Uri $asset -OutFile $TempFile
        # Run installer
        Start-Process -FilePath $TempFile -Wait
        # Remove installer
        Remove-Item $TempFile
    }
}