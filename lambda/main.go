package main

import (
    "context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "github.com/aws/aws-sdk-go-v2/service/ses"
    sesTypes     "github.com/aws/aws-sdk-go-v2/service/ses/types"

    "github.com/google/uuid"
)

var (
    TableName = os.Getenv("TABLE_NAME")
    FromEmail = os.Getenv("FROM_EMAIL")    // SESでVerify済み
    ToEmail   = os.Getenv("TO_EMAIL")    // 通知先メール
    Region    = os.Getenv("REGION")        // 東京リージョン例
    RecaptchaSecretKey = os.Getenv("RECAPTCHA_SECRET_KEY")
)

type ContactRequest struct {
    Name    string `json:"name"`
    Email   string `json:"email"`
    Subject   string `json:"subject"`
    Message string `json:"message"`
    RecaptchaToken string `json:"recaptchaToken"`
}

// --- reCAPTCHA のレスポンス構造体 ---
type RecaptchaResponse struct {
	Success bool `json:"success"`
}


// --- reCAPTCHA 検証 ---
func verifyRecaptcha(token string) bool {
	resp, err := http.PostForm(
		"https://www.google.com/recaptcha/api/siteverify",
		url.Values{
			"secret":   {RecaptchaSecretKey},
			"response": {token},
		},
	)
	if err != nil {
		log.Println("reCAPTCHA リクエストエラー:", err)
		return false
	}
	defer resp.Body.Close()

	var result RecaptchaResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Success
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(Region))
    if err != nil {
        return errorResponse("AWS設定の読み込みに失敗しました", 500)
    }

    dynamoDB := dynamodb.NewFromConfig(cfg)
    sesClient := ses.NewFromConfig(cfg)

    // JSONパース
    var data ContactRequest
    if err := json.Unmarshal([]byte(request.Body), &data); err != nil {
        return errorResponse("リクエストの解析に失敗しました", 400)
    }

    // 💡 reCAPTCHA チェック
	if !verifyRecaptcha(data.RecaptchaToken) {
		return errorResponse("reCAPTCHA認証に失敗しました", 400)
	}

    recordID := uuid.New().String()
    receivedAt := time.Now().UTC().Format(time.RFC3339)

    _, err = dynamoDB.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String(TableName),
        Item: map[string]dynamodbTypes.AttributeValue{
            "id":          &dynamodbTypes.AttributeValueMemberS{Value: recordID},
            "name":        &dynamodbTypes.AttributeValueMemberS{Value: data.Name},
            "email":       &dynamodbTypes.AttributeValueMemberS{Value: data.Email},
            "subject":     &dynamodbTypes.AttributeValueMemberS{Value: data.Subject},
            "message":     &dynamodbTypes.AttributeValueMemberS{Value: data.Message},
            "received_at": &dynamodbTypes.AttributeValueMemberS{Value: receivedAt},
        },
    })
    if err != nil {
        return errorResponse("DynamoDB保存に失敗しました", 500)
    }

    // SESメール通知
    _, err = sesClient.SendEmail(ctx, &ses.SendEmailInput{
        Source: aws.String(FromEmail),
        Destination: &sesTypes.Destination{
            ToAddresses: []string{ToEmail},
        },
        Message: &sesTypes.Message{
            Subject: &sesTypes.Content{
                Data: aws.String(fmt.Sprintf("【お問い合わせ】%s", data.Subject)),
            },
            Body: &sesTypes.Body{
                Text: &sesTypes.Content{
                    Data: aws.String(fmt.Sprintf(
                        "お問い合わせを受け付けました。\n\n"+
                            "■ 名前：%s\n"+
                            "■ メール：%s\n"+
                            "■ 件名：%s\n"+
                            "■ 送信日時：%s\n\n"+
                            "--- メッセージ ---\n%s\n",
                        data.Name, data.Email, data.Subject, receivedAt, data.Message,
                    )),
                },
            },
        },
        ReplyToAddresses: []string{data.Email},
    })
    if err != nil {
        return errorResponse("メール送信に失敗しました", 500)
    }

    return events.APIGatewayProxyResponse{
        StatusCode: 200,
        Body:       `{"message":"お問い合わせを受け付けました。"}`,
        Headers: map[string]string{
            "Content-Type": "application/json",
            "Access-Control-Allow-Origin": "*", // TODO: 本番環境では必要
        },
    }, nil
}

func errorResponse(msg string, code int) (events.APIGatewayProxyResponse, error) {
    return events.APIGatewayProxyResponse{
        StatusCode: code,
        Headers: map[string]string{
			"Content-Type": "application/json",
			"Access-Control-Allow-Origin": "*", // TODO: 本番環境では必要
		},
        Body:       fmt.Sprintf(`{"error": "%s"}`, msg),
    }, nil
}

func main() {
    lambda.Start(handler)
}
