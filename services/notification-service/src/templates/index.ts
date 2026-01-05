import { NotificationType } from '../types';

interface Template {
  subject: string;
  body: string;
  smsBody?: string;
}

const templates: Record<NotificationType, Template> = {
  [NotificationType.TRANSFER_CREATED]: {
    subject: 'Transfer Initiated - {{amount}} {{currency}} to {{recipientName}}',
    body: `
      <h2>Transfer Initiated</h2>
      <p>Hi {{senderName}},</p>
      <p>Your transfer of <strong>{{amount}} {{currency}}</strong> to {{recipientName}} has been created.</p>
      <p><strong>Transfer ID:</strong> {{transferId}}</p>
      <p><strong>Exchange Rate:</strong> 1 {{sourceCurrency}} = {{exchangeRate}} {{targetCurrency}}</p>
      <p><strong>Recipient Gets:</strong> {{recipientAmount}} {{targetCurrency}}</p>
      <p>Please complete the payment to proceed with your transfer.</p>
      <p>Thank you for using Movra!</p>
    `,
    smsBody: 'Movra: Transfer of {{amount}} {{currency}} to {{recipientName}} created. ID: {{transferId}}',
  },

  [NotificationType.TRANSFER_FUNDS_RECEIVED]: {
    subject: 'Funds Received - Processing Your Transfer',
    body: `
      <h2>Funds Received</h2>
      <p>Hi {{senderName}},</p>
      <p>We've received your payment of <strong>{{amount}} {{currency}}</strong>.</p>
      <p>Your transfer to {{recipientName}} is now being processed.</p>
      <p><strong>Transfer ID:</strong> {{transferId}}</p>
      <p>We'll notify you once the transfer is complete.</p>
    `,
    smsBody: 'Movra: Payment received. Transfer {{transferId}} to {{recipientName}} is processing.',
  },

  [NotificationType.TRANSFER_COMPLETED]: {
    subject: 'Transfer Complete - {{amount}} {{currency}} Sent!',
    body: `
      <h2>Transfer Complete!</h2>
      <p>Hi {{senderName}},</p>
      <p>Great news! Your transfer has been completed successfully.</p>
      <p><strong>Amount Sent:</strong> {{amount}} {{currency}}</p>
      <p><strong>Recipient:</strong> {{recipientName}}</p>
      <p><strong>Amount Received:</strong> {{recipientAmount}} {{targetCurrency}}</p>
      <p><strong>Transfer ID:</strong> {{transferId}}</p>
      <p>Thank you for choosing Movra!</p>
    `,
    smsBody: 'Movra: Transfer complete! {{recipientName}} received {{recipientAmount}} {{targetCurrency}}. ID: {{transferId}}',
  },

  [NotificationType.TRANSFER_FAILED]: {
    subject: 'Transfer Failed - Action Required',
    body: `
      <h2>Transfer Failed</h2>
      <p>Hi {{senderName}},</p>
      <p>Unfortunately, your transfer to {{recipientName}} could not be completed.</p>
      <p><strong>Transfer ID:</strong> {{transferId}}</p>
      <p><strong>Reason:</strong> {{failureReason}}</p>
      <p>Please contact support if you need assistance.</p>
    `,
    smsBody: 'Movra: Transfer {{transferId}} failed. Reason: {{failureReason}}. Contact support.',
  },

  [NotificationType.PICKUP_CODE_READY]: {
    subject: 'Cash Pickup Code Ready - {{transferId}}',
    body: `
      <h2>Your Cash Pickup Code is Ready</h2>
      <p>Hi {{recipientName}},</p>
      <p>You have a cash pickup waiting for you!</p>
      <p><strong>Amount:</strong> {{amount}} {{currency}}</p>
      <p><strong>Pickup Code:</strong> <span style="font-size: 24px; font-weight: bold;">{{pickupCode}}</span></p>
      <p><strong>Expires:</strong> {{expiresAt}}</p>
      <p>Present this code and your ID at any authorized pickup location.</p>
    `,
    smsBody: 'Movra: Cash pickup ready! Code: {{pickupCode}}. Amount: {{amount}} {{currency}}. Expires: {{expiresAt}}',
  },

  [NotificationType.WELCOME]: {
    subject: 'Welcome to Movra!',
    body: `
      <h2>Welcome to Movra!</h2>
      <p>Hi {{userName}},</p>
      <p>Thank you for joining Movra. We're excited to help you send money internationally.</p>
      <p>Get started by completing your profile verification to unlock all features.</p>
      <p>If you have any questions, our support team is here to help.</p>
    `,
    smsBody: 'Welcome to Movra, {{userName}}! Complete verification to start sending money internationally.',
  },

  [NotificationType.PASSWORD_RESET]: {
    subject: 'Password Reset Request',
    body: `
      <h2>Password Reset</h2>
      <p>Hi {{userName}},</p>
      <p>We received a request to reset your password.</p>
      <p>Click the link below to reset your password:</p>
      <p><a href="{{resetLink}}">Reset Password</a></p>
      <p>This link expires in {{expiresIn}}.</p>
      <p>If you didn't request this, please ignore this email.</p>
    `,
    smsBody: 'Movra: Password reset requested. Link expires in {{expiresIn}}.',
  },

  // Placeholder templates for other types
  [NotificationType.TRANSFER_AWAITING_FUNDS]: {
    subject: 'Awaiting Payment - {{transferId}}',
    body: '<p>Please complete payment for transfer {{transferId}}</p>',
    smsBody: 'Movra: Complete payment for transfer {{transferId}}',
  },
  [NotificationType.TRANSFER_PROCESSING]: {
    subject: 'Transfer Processing - {{transferId}}',
    body: '<p>Your transfer {{transferId}} is being processed</p>',
    smsBody: 'Movra: Transfer {{transferId}} processing',
  },
  [NotificationType.TRANSFER_REFUNDED]: {
    subject: 'Transfer Refunded - {{transferId}}',
    body: '<p>Your transfer {{transferId}} has been refunded</p>',
    smsBody: 'Movra: Transfer {{transferId}} refunded',
  },
  [NotificationType.PICKUP_REMINDER]: {
    subject: 'Reminder: Cash Pickup Expiring Soon',
    body: '<p>Your pickup code {{pickupCode}} expires on {{expiresAt}}</p>',
    smsBody: 'Movra: Pickup code {{pickupCode}} expires {{expiresAt}}',
  },
  [NotificationType.PICKUP_COLLECTED]: {
    subject: 'Cash Collected',
    body: '<p>Cash pickup {{transferId}} has been collected</p>',
    smsBody: 'Movra: Cash collected for {{transferId}}',
  },
  [NotificationType.KYC_APPROVED]: {
    subject: 'Verification Approved',
    body: '<p>Your identity verification has been approved!</p>',
    smsBody: 'Movra: Your verification is approved!',
  },
  [NotificationType.KYC_REJECTED]: {
    subject: 'Verification Needs Attention',
    body: '<p>Please update your verification documents</p>',
    smsBody: 'Movra: Verification needs attention',
  },
  [NotificationType.LOGIN_ALERT]: {
    subject: 'New Login Detected',
    body: '<p>New login to your account from {{device}} at {{location}}</p>',
    smsBody: 'Movra: New login from {{device}}',
  },
};

export function renderTemplate(
  type: NotificationType,
  data: Record<string, string>,
  channel: 'email' | 'sms' = 'email'
): { subject: string; body: string } {
  const template = templates[type];
  if (!template) {
    return { subject: 'Notification', body: 'You have a new notification from Movra.' };
  }

  const body = channel === 'sms' && template.smsBody ? template.smsBody : template.body;

  return {
    subject: interpolate(template.subject, data),
    body: interpolate(body, data),
  };
}

function interpolate(text: string, data: Record<string, string>): string {
  return text.replace(/\{\{(\w+)\}\}/g, (_, key) => data[key] || '');
}

export { templates };
