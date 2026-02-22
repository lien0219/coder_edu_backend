<#
.SYNOPSIS
    Coder Edu Backend - 一键部署脚本
.DESCRIPTION
    用法: powershell -ExecutionPolicy Bypass -File deploy.ps1
    配置: 复制 deploy.env.example 为 deploy.env 并填入真实值
#>

# 读取配置文件
$envFile = Join-Path $PSScriptRoot "deploy.env"
if (-not (Test-Path $envFile)) {
    Write-Host "未找到 deploy.env 配置文件！" -ForegroundColor Red
    Write-Host "请先复制 deploy.env.example 为 deploy.env 并填入真实值" -ForegroundColor Yellow
    exit 1
}

# 解析配置
$config = @{}
Get-Content $envFile | ForEach-Object {
    $line = $_.Trim()
    if ($line -and (-not $line.StartsWith("#")) -and $line.Contains("=")) {
        $idx = $line.IndexOf("=")
        $key = $line.Substring(0, $idx).Trim()
        $val = $line.Substring($idx + 1).Trim()
        $config[$key] = $val
    }
}

$Server = $config["DEPLOY_SERVER"]
$RemotePath = $config["DEPLOY_PATH"]
$ServiceName = $config["DEPLOY_SERVICE"]
$HealthUrl = $config["HEALTH_CHECK_URL"]

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Coder Edu Backend 一键部署" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# ===== 本地编译 =====
Write-Host "[1/4] 编译 Linux 版本..." -ForegroundColor Yellow
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
& go build -o coder_edu_backend "."

if ($LASTEXITCODE -ne 0) {
    Write-Host "编译失败！" -ForegroundColor Red
    exit 1
}
Write-Host "编译成功！" -ForegroundColor Green

# ===== 上传到服务器 =====
Write-Host ""
Write-Host "[2/4] 上传到服务器..." -ForegroundColor Yellow
$dest = "${Server}:${RemotePath}/coder_edu_backend.new"
& scp -o ConnectTimeout=10 coder_edu_backend $dest

if ($LASTEXITCODE -ne 0) {
    Write-Host "上传失败！" -ForegroundColor Red
    exit 1
}
Write-Host "上传成功！" -ForegroundColor Green

# ===== 远程替换并重启 =====
Write-Host ""
Write-Host "[3/4] 替换文件并重启服务..." -ForegroundColor Yellow
$cmd1 = "cd " + $RemotePath + " ; cp coder_edu_backend coder_edu_backend.bak ; systemctl stop " + $ServiceName + " ; mv coder_edu_backend.new coder_edu_backend ; chmod +x coder_edu_backend ; systemctl start " + $ServiceName + " ; sleep 2 ; systemctl is-active " + $ServiceName
& ssh -o ConnectTimeout=10 $Server $cmd1

if ($LASTEXITCODE -ne 0) {
    Write-Host "重启失败！正在回滚..." -ForegroundColor Red
    $cmd2 = "cd " + $RemotePath + " ; mv coder_edu_backend.bak coder_edu_backend ; systemctl restart " + $ServiceName
    & ssh $Server $cmd2
    Write-Host "已回滚到上一个版本" -ForegroundColor Yellow
    exit 1
}
Write-Host "服务重启成功！" -ForegroundColor Green

# ===== 健康检查 =====
Write-Host ""
Write-Host "[4/4] 健康检查..." -ForegroundColor Yellow
Start-Sleep -Seconds 3

$healthOk = $false
try {
    $resp = Invoke-RestMethod -Uri $HealthUrl -TimeoutSec 10
    if ($resp.code -eq 200) {
        $healthOk = $true
    }
} catch {
    Write-Host "健康检查请求异常" -ForegroundColor Yellow
}

if ($healthOk) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "  部署成功！" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    $cmd3 = "rm -f " + $RemotePath + "/coder_edu_backend.bak"
    & ssh $Server $cmd3
} else {
    Write-Host "请手动确认服务状态" -ForegroundColor Yellow
}

# 清理本地编译产物和还原环境变量
Remove-Item "coder_edu_backend" -ErrorAction SilentlyContinue
Remove-Item env:GOOS -ErrorAction SilentlyContinue
Remove-Item env:GOARCH -ErrorAction SilentlyContinue
Remove-Item env:CGO_ENABLED -ErrorAction SilentlyContinue
Write-Host ""
