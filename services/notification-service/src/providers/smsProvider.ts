import { config } from '../config';
import { NotificationProvider, SendResult } from './types';
import pino from 'pino';

const logger = pino({ name: 'sms-provider' });

export class SmsProvider implements NotificationProvider {
  name = 'sms';

  async send(recipient: string, _subject: string, body: string): Promise<SendResult> {
    // Simulated SMS sending - in production, integrate with Twilio/Nexmo/etc.
    if (config.sms.provider === 'simulated') {
      logger.info({ recipient, body: body.substring(0, 50) }, 'Simulated SMS sent');
      return {
        success: true,
        messageId: `SMS_${Date.now()}`,
      };
    }

    // Placeholder for real SMS provider integration
    // Example with Twilio:
    // const client = twilio(config.sms.accountSid, config.sms.authToken);
    // const message = await client.messages.create({
    //   body,
    //   from: config.sms.fromNumber,
    //   to: recipient,
    // });

    return {
      success: false,
      error: `SMS provider '${config.sms.provider}' not implemented`,
    };
  }
}
