import nodemailer from 'nodemailer';
import { config } from '../config';
import { NotificationProvider, SendResult } from './types';

export class EmailProvider implements NotificationProvider {
  name = 'email';
  private transporter: nodemailer.Transporter;

  constructor() {
    this.transporter = nodemailer.createTransport({
      host: config.smtp.host,
      port: config.smtp.port,
      secure: config.smtp.secure,
      auth: config.smtp.auth,
    });
  }

  async send(recipient: string, subject: string, body: string): Promise<SendResult> {
    try {
      const info = await this.transporter.sendMail({
        from: config.smtp.from,
        to: recipient,
        subject,
        html: body,
      });

      return {
        success: true,
        messageId: info.messageId,
      };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }
}
