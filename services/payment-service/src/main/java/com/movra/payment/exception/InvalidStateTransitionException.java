package com.movra.payment.exception;

import com.movra.payment.model.TransferStatus;

public class InvalidStateTransitionException extends RuntimeException {
    private final TransferStatus currentStatus;
    private final TransferStatus targetStatus;

    public InvalidStateTransitionException(TransferStatus current, TransferStatus target) {
        super(String.format("Invalid state transition: %s -> %s", current, target));
        this.currentStatus = current;
        this.targetStatus = target;
    }

    public TransferStatus getCurrentStatus() { return currentStatus; }
    public TransferStatus getTargetStatus() { return targetStatus; }
}
