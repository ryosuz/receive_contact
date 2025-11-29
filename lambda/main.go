package main

import (
    "context"
    "encoding/json"
    "fmt"
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

const (
    TableName = os.Getenv("TABLE_NAME")
    FromEmail = os.Getenv("FROM_EMAIL")    // SESでVerify済み
    ToEmail   = os.Getenv("TO_EMAIL")    // 通知先メール
    Region    = os.Getenv("REGION")        // 東京リージョン例
)

type ContactRequest struct {
    Name    string `json:"name"`
    Email   string `json:"email"`
    Title   string `json:"title"`
    Message string `json:"message"`
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(Region))
    if err != nil {
        return errorResponse("Failed to load AWS config", 500)
    }

    dynamoDB := dynamodb.NewFromConfig(cfg)
    sesClient := ses.NewFromConfig(cfg)

    var data ContactRequest
    if err := json.Unmarshal([]byte(request.Body), &data); err != nil {
        return errorResponse("Invalid request", 400)
    }

    recordID := uuid.New().String()
    receivedAt := time.Now().UTC().Format(time.RFC3339)

    _, err = dynamoDB.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String(TableName),
        Item: map[string]dynamodbTypes.AttributeValue{  // ← 修正！
            "id":          &dynamodbTypes.AttributeValueMemberS{Value: recordID},
            "name":        &dynamodbTypes.AttributeValueMemberS{Value: data.Name},
            "email":       &dynamodbTypes.AttributeValueMemberS{Value: data.Email},
            "title":       &dynamodbTypes.AttributeValueMemberS{Value: data.Title},
            "message":     &dynamodbTypes.AttributeValueMemberS{Value: data.Message},
            "received_at": &dynamodbTypes.AttributeValueMemberS{Value: receivedAt},
        },
    })
    if err != nil {
        return errorResponse("Failed to save to DynamoDB", 500)
    }

    _, err = sesClient.SendEmail(ctx, &ses.SendEmailInput{
        Source: aws.String(FromEmail),
        Destination: &sesTypes.Destination{  // ← 修正！
            ToAddresses: []string{ToEmail},
        },
        Message: &sesTypes.Message{          // ← 修正！
            Subject: &sesTypes.Content{
                Data: aws.String(fmt.Sprintf("【お問い合わせ】%s", data.Title)),
            },
            Body: &sesTypes.Body{
                Text: &sesTypes.Content{
                    Data: aws.String(fmt.Sprintf(
                        "お問い合わせを受け付けました。\n\n"+
                            "■ 名前：%s\n"+
                            "■ メール：%s\n"+
                            "■ タイトル：%s\n"+
                            "■ 送信日時：%s\n\n"+
                            "--- メッセージ ---\n%s\n",
                        data.Name, data.Email, data.Title, receivedAt, data.Message,
                    )),
                },
            },
        },
        ReplyToAddresses: []string{data.Email},
    })
    if err != nil {
        return errorResponse("Failed to send email", 500)
    }

    return events.APIGatewayProxyResponse{
        StatusCode: 200,
        Body:       `{"message":"お問い合わせを受け付けました。"}`,
        Headers: map[string]string{
            "Content-Type": "application/json",
        },
    }, nil
}

func errorResponse(msg string, code int) (events.APIGatewayProxyResponse, error) {
    return events.APIGatewayProxyResponse{
        StatusCode: code,
        Body:       fmt.Sprintf(`{"error": "%s"}`, msg),
    }, nil
}

func main() {
    lambda.Start(handler)
}
