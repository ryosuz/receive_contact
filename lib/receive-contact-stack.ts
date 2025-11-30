import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';

export class ReceiveContactStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // DynamoDB テーブル
    const table = new dynamodb.Table(this, 'ContactMessages', {
      tableName: 'contact_messages',
      partitionKey: { name: 'id', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'received_at', type: dynamodb.AttributeType.STRING },
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // Go Lambda 関数
    const lambdaFunc = new lambda.Function(this, 'ContactLambda', {
      runtime: lambda.Runtime.PROVIDED_AL2,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset('lambda'), // main.goがビルド済み
      architecture: lambda.Architecture.ARM_64,
      environment: {
        TABLE_NAME: table.tableName,
        FROM_EMAIL: 'contact@ryosuz.com',
        TO_EMAIL: 'contact@ryosuz.com',
        REGION: this.region
      },
    });

    // DynamoDB アクセス権限
    table.grantWriteData(lambdaFunc);

    // SES 送信権限
    lambdaFunc.addToRolePolicy(
      new cdk.aws_iam.PolicyStatement({
        actions: ["ses:SendEmail", "ses:SendRawEmail"],
        resources: ["*"]
      })
    );

    // API Gateway（POST /）
    const api = new apigateway.LambdaRestApi(this, 'ContactApi', {
      handler: lambdaFunc,
      proxy: false,
      defaultCorsPreflightOptions: {
        allowOrigins: ['*'],
        allowMethods: ['POST'],
      },
    });
    const contact = api.root.addResource('contact');
    contact.addCorsPreflight({
      allowOrigins: [
        "https://portfolio.ryosuz.com",
        "http://localhost:3000"
      ],
      allowMethods: ['POST'],
    });
    contact.addMethod('POST');
  }
}
