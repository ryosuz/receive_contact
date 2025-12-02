# receive_contact

AWSサーバーレス構成で問い合わせを受付・保存・通知するAPIです。詳細仕様・設計は `docs/SPECIFICATION.md` にまとめています。

## 特徴

- API Gateway → Lambda(Go) → DynamoDB → SES で問い合わせを処理
- reCAPTCHA v2 によるボット対策
- AWS CDK でIaC管理、GitHub ActionsでCI/CD

## 技術スタック（抜粋）

| カテゴリ | 技術 |
|----------|------|
| 言語 | Go 1.22 |
| サーバーレス | AWS Lambda (PROVIDED_AL2, ARM64) |
| ストレージ | Amazon DynamoDB |
| 通知 | Amazon SES |
| API | Amazon API Gateway (REST) |
| IaC / CI | AWS CDK, GitHub Actions |

## 開発 & デプロイ概要

1. `lambda/` 直下で Go Lambda をビルド（`GOOS=linux GOARCH=arm64`）
2. 生成した `bootstrap` を含めて CDK デプロイ
3. GitHub Actions（`main` ブランチ）で自動デプロイ可能

より詳しい手順・CI設定・API仕様は **docs/SPECIFICATION.md** を参照してください。

## ディレクトリ構成

```
receive_contact/
├── lambda/                  # Go Lambda ソース
├── lib/                     # CDK スタック
├── bin/                     # CDK エントリポイント
├── docs/
│   └── SPECIFICATION.md     # 仕様・設計ドキュメント
├── .github/workflows/       # CI/CD
└── README.md
```

## ドキュメント

- 詳細仕様・API定義・データ設計: [`docs/SPECIFICATION.md`](docs/SPECIFICATION.md)

README は概要紹介に留め、実装や運用の詳細は SPECIFICATION を参照してください。
