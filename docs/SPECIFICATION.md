# 問い合わせ受付API 仕様・設計書

## 1. 概要

Webフォームからの問い合わせを受け付け、データベースへ保存し、管理者へメール通知を行うサーバーレスAPIです。

## 2. システム構成

```
┌─────────────────┐
│   Webフォーム    │
│ (portfolio.ryosuz.com)
└────────┬────────┘
         │ POST /contact
         ▼
┌─────────────────┐
│  API Gateway    │
│  (REST API)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Lambda (Go)    │
│  - reCAPTCHA検証│
│  - バリデーション│
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌────────┐ ┌────────┐
│DynamoDB│ │  SES   │
│ 保存   │ │ 通知   │
└────────┘ └────────┘
```

## 3. 技術スタック

| 技術 | バージョン | 用途 |
|------|-----------|------|
| Go | 1.22 | Lambda関数実装 |
| AWS Lambda | PROVIDED_AL2 (ARM64) | 問い合わせ処理 |
| API Gateway | REST API | HTTPエンドポイント |
| DynamoDB | - | 問い合わせデータ保存 |
| SES | - | メール通知 |
| AWS CDK | - | IaC（インフラ定義） |
| GitHub Actions | - | CI/CD |

## 4. API仕様

### 4.1 エンドポイント

| メソッド | パス | 説明 |
|---------|------|------|
| POST | /contact | 問い合わせ送信 |

### 4.2 リクエスト

**Content-Type:** `multipart/form-data`

| フィールド名 | 型 | 必須 | 制限 | 説明 |
|-------------|-----|------|------|------|
| name | string | ✓ | - | 送信者名 |
| email | string | ✓ | 256文字以下 | 送信者メールアドレス |
| subject | string | ✓ | - | 件名 |
| message | string | ✓ | 2000文字以下 | 問い合わせ内容 |
| recaptchaToken | string | ✓ | - | reCAPTCHA v2トークン |

### 4.3 レスポンス

#### 成功時 (200)
```json
{
  "message": "お問い合わせを受け付けました。"
}
```

#### エラー時 (400/500)
```json
{
  "error": "エラーメッセージ"
}
```

### 4.4 エラーコード

| コード | メッセージ | 原因 |
|--------|-----------|------|
| 400 | リクエストの解析に失敗しました | 不正なリクエスト形式 |
| 400 | reCAPTCHA認証に失敗しました | reCAPTCHA検証失敗 |
| 500 | AWS設定の読み込みに失敗しました | AWS SDK初期化エラー |
| 500 | DynamoDB保存に失敗しました | DB書き込みエラー |
| 500 | メール送信に失敗しました | SES送信エラー |

### 4.5 CORS設定

| 設定 | 値 |
|------|-----|
| 許可オリジン | `https://portfolio.ryosuz.com`, `http://localhost:4321` |
| 許可メソッド | POST |

## 5. データ設計

### 5.1 DynamoDB テーブル

**テーブル名:** `contact_messages`

| 属性名 | 型 | キー | 説明 |
|--------|-----|------|------|
| id | String | Partition Key | UUID v4 |
| received_at | String | Sort Key | 受信日時 (RFC3339) |
| name | String | - | 送信者名 |
| email | String | - | 送信者メールアドレス |
| subject | String | - | 件名 |
| message | String | - | 問い合わせ内容 |

## 6. 処理フロー

```
1. リクエスト受信
   └─ multipart/form-data をパース
   
2. バリデーション
   ├─ 必須項目チェック (name, email, subject, message, recaptchaToken)
   ├─ email: 256文字以下
   └─ message: 2000文字以下

3. reCAPTCHA検証
   └─ Google reCAPTCHA API へトークン検証

4. DynamoDB保存
   └─ UUID生成 → レコード保存

5. SES通知
   └─ 管理者へメール送信 (ReplyTo: 送信者メール)

6. レスポンス返却
```

## 7. 環境変数

| 変数名 | 説明 | 例 |
|--------|------|-----|
| TABLE_NAME | DynamoDBテーブル名 | contact_messages |
| FROM_EMAIL | 送信元メールアドレス (SES検証済み) | contact@ryosuz.com |
| TO_EMAIL | 通知先メールアドレス | contact@ryosuz.com |
| REGION | AWSリージョン | ap-northeast-1 |
| RECAPTCHA_SECRET_KEY | reCAPTCHA シークレットキー | - |

## 8. セキュリティ

- **reCAPTCHA v2**: ボット対策
- **入力バリデーション**: 文字数制限によるDoS対策
- **CORS制限**: 許可オリジンのみアクセス可能
- **IAM最小権限**: Lambda は DynamoDB書き込み・SES送信のみ許可

## 9. ディレクトリ構成

```
receive_contact/
├── .github/workflows/
│   └── deploy.yml          # CI/CD設定
├── bin/
│   └── project-root.ts     # CDKエントリポイント
├── lambda/
│   ├── main.go             # Lambda関数
│   ├── go.mod
│   └── go.sum
├── lib/
│   └── receive-contact-stack.ts  # CDKスタック定義
├── test/
│   └── receive_contact.test.ts
├── cdk.json
├── package.json
└── README.md
```

## 10. デプロイ

### 10.1 手動デプロイ

```bash
# Lambdaビルド
cd lambda
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap main.go

# CDKデプロイ
cd ..
cdk deploy
```

### 10.2 CI/CD (GitHub Actions)

`main` ブランチへのプッシュで自動デプロイ

**必要なSecrets:**
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `RECAPTCHA_SECRET_KEY`

## 11. メール通知フォーマット

**件名:** `【お問い合わせ】{subject}`

**本文:**
```
お問い合わせを受け付けました。

■ 名前：{name}
■ メール：{email}
■ 件名：{subject}
■ 送信日時：{received_at}

--- メッセージ ---
{message}
```

**ReplyTo:** 送信者のメールアドレス
