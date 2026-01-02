package com.movra.payment.model;

public enum TransferStatus {
    CREATED,
    AWAITING_FUNDS,
    FUNDS_RECEIVED,
    CONVERTING,
    CONVERTED,
    PAYOUT_PENDING,
    PAYOUT_PROCESSING,
    COMPLETED,
    FAILED,
    REFUNDED,
    CANCELLED;

    public boolean canTransitionTo(TransferStatus newStatus) {
        return switch (this) {
            case CREATED -> newStatus == AWAITING_FUNDS || newStatus == CANCELLED;
            case AWAITING_FUNDS -> newStatus == FUNDS_RECEIVED || newStatus == CANCELLED || newStatus == FAILED;
            case FUNDS_RECEIVED -> newStatus == CONVERTING || newStatus == FAILED || newStatus == REFUNDED;
            case CONVERTING -> newStatus == CONVERTED || newStatus == FAILED || newStatus == REFUNDED;
            case CONVERTED -> newStatus == PAYOUT_PENDING || newStatus == FAILED || newStatus == REFUNDED;
            case PAYOUT_PENDING -> newStatus == PAYOUT_PROCESSING || newStatus == FAILED || newStatus == REFUNDED;
            case PAYOUT_PROCESSING -> newStatus == COMPLETED || newStatus == FAILED || newStatus == REFUNDED;
            case COMPLETED, FAILED, REFUNDED, CANCELLED -> false;
        };
    }
}
