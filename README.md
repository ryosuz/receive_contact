
---

## 🐹 使用技術

| 技術 | 使用 |
|------|------|
| Lambda | Go 1.22 + SDK v2 |
| DynamoDB | 問い合わせ保存 |
| SES | メール通知 |
| API Gateway | エンドポイント |
| CDK | インフラ構築 |
| CI/CD | GitHub Actions |
| Runtime | `provided.al2 (arm64)` |

---

## 🧩 ディレクトリ構成

project-root/
├── lambda/
│ ├── main.go
│ └── go.mod
├── lib/
│ └── contact-form-stack.ts
├── bin/
│ └── project-root.ts
├── .github/workflows/
│ └── deploy.yml
├── build.ps1 # Windows用ビルド＆デプロイスクリプト
├── cdk.json
└── README.md

---

## 🛠️ Lambda ビルド方法（ローカル）

```powershell
cd lambda
$env:GOOS="linux"
$env:GOARCH="arm64"
$env:CGO_ENABLED="0"
go build -o bootstrap main.go
Compress-Archive bootstrap function.zip -Force
