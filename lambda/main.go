package main

import (
	"bytes"
    "context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
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

func parseMultipartContact(request events.APIGatewayProxyRequest) (ContactRequest, error) {
	body := []byte(request.Body)
	if request.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			return ContactRequest{}, err
		}
		body = decoded
	}

	contentType := getHeaderValue(request.Headers, "Content-Type")
	if contentType == "" {
		return ContactRequest{}, fmt.Errorf("content type header missing")
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ContactRequest{}, err
	}
	if !strings.EqualFold(mediaType, "multipart/form-data") {
		return ContactRequest{}, fmt.Errorf("unsupported media type")
	}
	boundary, ok := params["boundary"]
	if !ok {
		return ContactRequest{}, fmt.Errorf("boundary not found")
	}

	mr := multipart.NewReader(bytes.NewReader(body), boundary)
	result := ContactRequest{}

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ContactRequest{}, err
		}
		valueBytes, err := io.ReadAll(part)
		if err != nil {
			return ContactRequest{}, err
		}
		value := strings.TrimSpace(string(valueBytes))
		switch part.FormName() {
		case "name":
			result.Name = value
		case "email":
			result.Email = value
		case "subject":
			result.Subject = value
		case "message":
			result.Message = value
		case "recaptchaToken":
			result.RecaptchaToken = value
		}
	}

	if result.Name == "" || result.Email == "" || result.Subject == "" || result.Message == "" || result.RecaptchaToken == "" {
		return ContactRequest{}, fmt.Errorf("必要な項目が不足しています")
	}
	if 256 < len(result.email) {
		return ContactRequest{}, fmt.Errorf("不正なメールアドレスです")
	}
	if 2000 < len(result.message) {
		return ContactRequest{}, fmt.Errorf("メッセージが長すぎます")
	}

	return result, nil
}

func getHeaderValue(headers map[string]string, key string) string {
	if val, ok := headers[key]; ok {
		return val
	}
	for k, v := range headers {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
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

    data, err := parseMultipartContact(request)
    if err != nil {
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
