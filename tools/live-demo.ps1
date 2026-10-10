param([switch]$Resume)
$ErrorActionPreference='Stop'
if (-not (Test-Path -LiteralPath '.env')) { throw 'Create local .env from .env.example first.' }
if (-not (Test-Path -LiteralPath 'evidence/local/direct.json')) { throw 'Complete go run ./cmd/direct before running the FlowKit live demo.' }
$env:DASHBOARD_PORT='8081'
$env:LOAD_EXAMPLES='false'
docker compose -p flowkit-acceptance up -d --wait
if ($LASTEXITCODE -ne 0) { throw 'Acceptance Compose startup failed' }
$taskDemoArgs=@('run','./cmd/acceptance','-url','http://localhost:8081','-project','flowkit-acceptance')
if ($Resume) { $taskDemoArgs+='-resume' }
go @taskDemoArgs
if ($LASTEXITCODE -ne 0) { throw 'Live demo stopped; inspect evidence before sending more requests.' }
