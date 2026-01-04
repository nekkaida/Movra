package com.movra.payment.exception;

public class RateLockExpiredException extends RuntimeException {
    private final String transferId;

    public RateLockExpiredException(String transferId) {
        super("Rate lock has expired for transfer: " + transferId);
        this.transferId = transferId;
    }

    public String getTransferId() { return transferId; }
}
