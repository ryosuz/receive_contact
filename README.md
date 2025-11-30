# AWS サーバーレス お問い合わせフォーム構築手順  
**Go (Lambda) + DynamoDB + SES + API Gateway + CDK + GitHub Actions**

## 1. プロジェクト概要
お問い合わせフォームから送信された問い合わせ内容を  
**API Gateway → Lambda（Go）→ DynamoDB 保存 → SES でメール通知** するサーバーレス構成です。

## 2. システム構成図
```
[Webフォーム]
    ↓ (POST /contact)
API Gateway
    ↓
Lambda (Go)
 ┌──────────────┐
 │ DynamoDB 保存 │
 │ SES メール通知│
 └──────────────┘
```

## 3. 使用技術一覧
| 技術 | 用途 |
|------|------|
| Go 1.22 | Lambda 実装 |
| AWS Lambda | 問い合わせ処理 |
| API Gateway | 問い合わせエンドポイント |
| DynamoDB | データ保存 |
| SES | メール通知 |
| CDK | IaC |
| GitHub Actions | CI/CD |

## 4. ディレクトリ構成
```
project-root/
├── lambda/
│   ├── main.go
│   └── go.mod
├── lib/
│   └── receive-contact-stack.ts
├── bin/
│   └── project-root.ts
├── .github/workflows/
│   └── deploy.yml
├── build.ps1
├── cdk.json
└── README.md
```

## 5. Lambda（Go）コード抜粋
```go
var (
    TableName = os.Getenv("TABLE_NAME")
    FromEmail = os.Getenv("FROM_EMAIL")
    ToEmail   = os.Getenv("TO_EMAIL")
)
```

## 6. CDK スタック定義抜粋
```ts
const api = new apigateway.LambdaRestApi(this, 'ContactApi', {
  handler: lambdaFunc,
  proxy: false,
});
const contact = api.root.addResource('contact');
contact.addMethod('POST');
```

## 7. GitHub Actions（CI/CD）
```yaml
name: Deploy Lambda with CDK
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with: { go-version: '1.22' }
      - uses: actions/setup-node@v4
        with: { node-version: '18' }
      - run: npm install -g aws-cdk && npm install
      - run: |
          cd lambda
          GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap main.go
          zip function.zip bootstrap
          cd ..
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: ap-northeast-1
      - run: cdk deploy --require-approval never
```

## 8. デプロイ手順
```
aws configure
aws sts get-caller-identity
cdk bootstrap
cdk deploy
```

## 9. テスト
```bash
curl -X POST https://<API-ID>.execute-api.ap-northeast-1.amazonaws.com/prod/contact   -H "Content-Type: application/json"   -d '{"name":"test","email":"test@example.com","subject":"test","message":"hello"}'
```

## 10. まとめ
- Lambda 実装済み
- CDK によるIaC構築済み
- API Gateway連携済み
- DynamoDB保存処理確認済み
- SESメール通知確認済み
- CI/CD導入済み
- デプロイ＆疎通確認済み
