#Requires -Version 5.1
[CmdletBinding()]
param(
  [string]$RootPath = $PSScriptRoot,
  [string]$OutputFile = "MANIFEST.json",
  [switch]$IncludeHashes = $false,
  [string[]]$ExcludePatterns = @("*.log","*.tmp","node_modules",".git","bin","obj","vendor","MANIFEST.json",".DS_Store"),
  [switch]$AsTool = $false
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$PSDefaultParameterValues['Out-File:Encoding'] = 'utf8'

function Get-ReadableSize([long]$Bytes) {
  if ($Bytes -ge 1GB) { return "{0:N2} GB" -f ($Bytes / 1GB) }
  if ($Bytes -ge 1MB) { return "{0:N2} MB" -f ($Bytes / 1MB) }
  if ($Bytes -ge 1KB) { return "{0:N2} KB" -f ($Bytes / 1KB) }
  return "$Bytes bytes"
}

# Mapas de categoria
$CategoryMap = @{
  'Source'         = @('.go','.js','.ts','.py','.java','.cs','.rs','.cpp','.c','.h')
  'Config'         = @('.json','.yaml','.yml','.toml','.env','.ini','.conf')
  'Documentation'  = @('.md','.txt','.rst','.adoc')
  'Script'         = @('.ps1','.psm1','.sh','.bash','.bat','.cmd')
}

function Get-FileCategory([string]$Extension) {
  $ext = ''
  if ($null -ne $Extension) { 
    $ext = $Extension.ToLower() 
  }
  
  foreach ($k in $CategoryMap.Keys) { 
    if ($CategoryMap[$k] -contains $ext) { 
      return $k 
    } 
  }
  return 'Other'
}

function Should-Exclude([string]$FullPath) {
  foreach ($pattern in $ExcludePatterns) {
    if ($FullPath -like "*$pattern*") { return $true }
    
    $seg = [IO.Path]::GetFileName($pattern.Trim('*'))
    if ($seg) {
      $segments = $FullPath -split '[\\/]'
      foreach ($s in $segments) {
        if ($s.ToLower() -eq $seg.ToLower()) {
          return $true
        }
      }
    }
  }
  return $false
}

try {
  $sw = [System.Diagnostics.Stopwatch]::StartNew()
  $resolvedRoot = (Resolve-Path -LiteralPath $RootPath).Path

  Write-Host "🔍 Inventário: $resolvedRoot" -ForegroundColor Cyan
  Write-Host "📊 Coletando dados...`n" -ForegroundColor Yellow

  # Diretórios
  Write-Verbose "Coletando diretórios..."
  $dirs = Get-ChildItem -LiteralPath $resolvedRoot -Directory -Recurse -ErrorAction SilentlyContinue |
          Where-Object { -not (Should-Exclude $_.FullName) } |
          Sort-Object FullName

  # Arquivos
  Write-Verbose "Coletando arquivos..."
  $files = Get-ChildItem -LiteralPath $resolvedRoot -File -Recurse -ErrorAction SilentlyContinue |
           Where-Object { -not (Should-Exclude $_.FullName) } |
           Sort-Object FullName

  $filesByCategory = @{}
  $filesByExtension = @{}
  $totalSize = [long]0

  # Compatível com PS 5.1
  if ($PSVersionTable.PSVersion.Major -ge 6) {
    $fileEntries = New-Object System.Collections.Generic.List[object]
  } else {
    $fileEntries = New-Object System.Collections.ArrayList
  }

  Write-Verbose "Processando $($files.Count) arquivos..."
  $processedCount = 0
  
  foreach ($f in $files) {
    $processedCount++
    if ($processedCount % 100 -eq 0) {
      Write-Verbose "Processado: $processedCount/$($files.Count)"
    }
    
    $rel = ($f.FullName.Substring($resolvedRoot.Length)).TrimStart('\','/')
    
    $ext = ''
    if ($null -ne $f.Extension) { 
      $ext = $f.Extension.ToLower() 
    }
    
    $cat = Get-FileCategory $ext
    $size = [long]$f.Length
    $totalSize += $size

    $hashVal = $null
    if ($IncludeHashes) {
      try { 
        $hashVal = (Get-FileHash -LiteralPath $f.FullName -Algorithm SHA256).Hash 
      } catch { 
        $hashVal = $null 
      }
    }

    $entry = [ordered]@{
      path                  = $rel
      name                  = $f.Name
      extension             = $ext
      category              = $cat
      size_bytes            = $size
      size_readable         = Get-ReadableSize $size
      last_write_time_utc   = $f.LastWriteTimeUtc.ToString("o")
    }
    
    if ($IncludeHashes -and $hashVal) { 
      $entry.hash_sha256 = $hashVal 
    }

    [void]$fileEntries.Add($entry)

    if (-not $filesByCategory.ContainsKey($cat)) { 
      $filesByCategory[$cat] = @{ count=0; total_size=[long]0 } 
    }
    $filesByCategory[$cat].count++
    $filesByCategory[$cat].total_size += $size

    if (-not $filesByExtension.ContainsKey($ext)) { 
      $filesByExtension[$ext] = @{ count=0; total_size=[long]0 } 
    }
    $filesByExtension[$ext].count++
    $filesByExtension[$ext].total_size += $size
  }

  Write-Verbose "Construindo manifesto..."
  
  $manifest = [ordered]@{
    schema_version      = "1.0.0"
    metadata            = [ordered]@{
      generated_at_utc  = (Get-Date).ToUniversalTime().ToString("o")
      root_path         = $RootPath
      root_path_resolved= $resolvedRoot
      generator         = "Generate-Manifest.ps1@v1.2.3"
      environment       = @{
        os              = $env:OS
        pwsh_version    = $PSVersionTable.PSVersion.ToString()
        pwsh_edition    = $PSVersionTable.PSEdition
      }
    }
    summary = [ordered]@{
      total_files           = $files.Count
      total_directories     = $dirs.Count
      total_size_bytes      = $totalSize
      total_size_readable   = Get-ReadableSize $totalSize
      files_by_category     = @{}
      files_by_extension    = @{}
    }
    structure = [ordered]@{
      directories = @()
      files       = @()
    }
  }

  foreach ($d in $dirs) {
    $rel = ($d.FullName.Substring($resolvedRoot.Length)).TrimStart('\','/')
    $manifest.structure.directories += [ordered]@{ 
      path = $rel
      name = $d.Name 
    }
  }

  foreach ($k in ($filesByCategory.Keys | Sort-Object)) {
    $manifest.summary.files_by_category[$k] = [ordered]@{
      count               = $filesByCategory[$k].count
      total_size_bytes    = $filesByCategory[$k].total_size
      total_size_readable = Get-ReadableSize $filesByCategory[$k].total_size
    }
  }

  foreach ($k in ($filesByExtension.Keys | Sort-Object)) {
    $manifest.summary.files_by_extension[$k] = [ordered]@{
      count               = $filesByExtension[$k].count
      total_size_bytes    = $filesByExtension[$k].total_size
      total_size_readable = Get-ReadableSize $filesByExtension[$k].total_size
    }
  }

  $manifest.structure.files = @($fileEntries)

  Write-Verbose "Salvando JSON..."
  $outputPath = Join-Path $resolvedRoot $OutputFile
  $manifest | ConvertTo-Json -Depth 10 | Out-File -FilePath $outputPath -Encoding UTF8

  # Console summary
  Write-Host "═══════════════════════════════════════════════" -ForegroundColor Cyan
  Write-Host "             RESUMO DO INVENTÁRIO              " -ForegroundColor Cyan
  Write-Host "═══════════════════════════════════════════════" -ForegroundColor Cyan
  Write-Host ""
  Write-Host ("📁 Diretórios:  {0}" -f $manifest.summary.total_directories) -ForegroundColor Green
  Write-Host ("📄 Arquivos:    {0}" -f $manifest.summary.total_files) -ForegroundColor Green
  Write-Host ("💾 Tamanho:     {0}" -f $manifest.summary.total_size_readable) -ForegroundColor Green
  Write-Host ("⚡ Tempo:       {0:N2}s" -f $sw.Elapsed.TotalSeconds) -ForegroundColor Gray
  Write-Host ("🔧 PowerShell:  {0} {1}" -f $PSVersionTable.PSEdition, $PSVersionTable.PSVersion) -ForegroundColor Gray
  Write-Host ""
  Write-Host "📊 Por Categoria:" -ForegroundColor Yellow
  foreach ($cat in ($manifest.summary.files_by_category.Keys | Sort-Object)) {
    $c = $manifest.summary.files_by_category[$cat]
    Write-Host ("   {0}: {1} ({2})" -f $cat, $c.count, $c.total_size_readable) -ForegroundColor Cyan
  }
  Write-Host ""
  Write-Host "✅ Salvo em: $outputPath" -ForegroundColor Green
  Write-Host "═══════════════════════════════════════════════`n" -ForegroundColor Cyan

  if ($AsTool) {
    $result = @{
      ok      = $true
      summary = "Manifest gerado com sucesso"
      details = @{ manifest_path = $outputPath }
      metrics = @{ 
        elapsed_ms = $sw.ElapsedMilliseconds
        files = $manifest.summary.total_files 
      }
      audit   = @{ 
        actor = "agent"
        at = (Get-Date).ToUniversalTime().ToString("o") 
      }
    } | ConvertTo-Json -Depth 5
    Write-Output $result
  }

  $sw.Stop()
  return $outputPath
}
catch {
  Write-Error ("❌ Falha ao gerar manifest: {0}" -f $_.Exception.Message)
  
  if ($AsTool) {
    $err = @{
      ok      = $false
      summary = "Erro ao gerar manifest"
      details = @{ error = $_.Exception.Message }
    } | ConvertTo-Json -Depth 5
    Write-Output $err
  }
  
  exit 1
}
