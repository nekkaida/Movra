export interface SendResult {
  success: boolean;
  messageId?: string;
  error?: string;
}

export interface NotificationProvider {
  name: string;
  send(recipient: string, subject: string, body: string): Promise<SendResult>;
}
