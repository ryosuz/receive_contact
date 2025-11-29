#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { ReceiveContactStack } from '../lib/receive-contact-stack';

const app = new cdk.App();
new ReceiveContactStack(app, 'ReceiveContactStack', {
  env: {
    region: process.env.CDK_DEFAULT_REGION || 'ap-northeast-1',
    account: process.env.CDK_DEFAULT_ACCOUNT,
  },
});
