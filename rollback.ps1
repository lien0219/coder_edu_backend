# ============================================
# 一键回滚脚本 - Coder Edu Backend
# 用法: .\rollback.ps1
# ============================================

$ErrorActionPreference = "Stop"

# 读取配置文件
$envFile = Join-Path $PSScriptRoot "deploy.env"
if (-not (Test-Path $envFile)) {
    Write-Host "未找到 deploy.env 配置文件！" -ForegroundColor Red
    exit 1
}

$config = @{}
Get-Content $envFile | ForEach-Object {
    if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
        $config[$matches[1].Trim()] = $matches[2].Trim()
    }
}

$Server = $config["DEPLOY_SERVER"]
$RemotePath = $config["DEPLOY_PATH"]
$ServiceName = $config["DEPLOY_SERVICE"]

Write-Host ""
Write-Host "========================================" -ForegroundColor Yellow
Write-Host "  Coder Edu Backend - 回滚到上一版本" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow
Write-Host ""

$rollbackCmd = 'cd {0}; if [ -f coder_edu_backend.bak ]; then systemctl stop {1}; cp coder_edu_backend.bak coder_edu_backend; chmod +x coder_edu_backend; systemctl start {1}; sleep 2; echo rollback_done; systemctl is-active {1}; else echo no_backup; exit 1; fi' -f $RemotePath, $ServiceName
ssh -o ConnectTimeout=10 $Server $rollbackCmd

if ($LASTEXITCODE -ne 0) {
    Write-Host "回滚失败！请手动检查服务器" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "回滚完成！" -ForegroundColor Green
Write-Host ""
