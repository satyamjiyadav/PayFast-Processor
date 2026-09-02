ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'initiated';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'processed';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'settled';

ALTER TYPE payout_schedule ADD VALUE IF NOT EXISTS 'instant';
ALTER TYPE payout_schedule ADD VALUE IF NOT EXISTS '1_hour';
ALTER TYPE payout_schedule ADD VALUE IF NOT EXISTS '24_hours';
