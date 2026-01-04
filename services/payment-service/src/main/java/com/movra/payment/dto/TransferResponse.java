package com.movra.payment.dto;

import com.movra.payment.model.FundingMethod;
import com.movra.payment.model.PayoutMethod;
import com.movra.payment.model.TransferStatus;
import lombok.Builder;
import lombok.Data;

import java.math.BigDecimal;
import java.time.Instant;

@Data
@Builder
public class TransferResponse {
    private String id;
    private String userId;
    private TransferStatus status;

    private String sourceCurrency;
    private BigDecimal sourceAmount;
    private String targetCurrency;
    private BigDecimal targetAmount;

    private BigDecimal exchangeRate;
    private String rateLockId;
    private Instant rateExpiresAt;

    private BigDecimal feeAmount;
    private String feeCurrency;

    private FundingMethod fundingMethod;
    private PayoutMethod payoutMethod;

    private String recipientId;
    private RecipientResponse recipient;

    private FundingDetailsResponse fundingDetails;

    private Instant createdAt;
    private Instant updatedAt;
    private Instant completedAt;

    private String failureReason;
}
