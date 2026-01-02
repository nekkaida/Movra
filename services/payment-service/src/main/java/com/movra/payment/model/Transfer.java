package com.movra.payment.model;

import jakarta.persistence.*;
import lombok.Data;
import lombok.NoArgsConstructor;
import lombok.AllArgsConstructor;
import lombok.Builder;

import java.math.BigDecimal;
import java.time.Instant;
import java.util.UUID;

@Entity
@Table(name = "transfers")
@Data
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class Transfer {

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;

    @Column(nullable = false)
    private UUID userId;

    @Column(nullable = false, unique = true)
    private String idempotencyKey;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private TransferStatus status;

    // Source amount
    @Column(nullable = false, length = 3)
    private String sourceCurrency;

    @Column(nullable = false, precision = 19, scale = 4)
    private BigDecimal sourceAmount;

    // Target amount
    @Column(nullable = false, length = 3)
    private String targetCurrency;

    @Column(nullable = false, precision = 19, scale = 4)
    private BigDecimal targetAmount;

    // Exchange rate
    @Column(nullable = false, precision = 19, scale = 8)
    private BigDecimal exchangeRate;

    private String rateLockId;

    private Instant rateExpiresAt;

    // Fee
    @Column(nullable = false, precision = 19, scale = 4)
    private BigDecimal feeAmount;

    @Column(nullable = false, length = 3)
    private String feeCurrency;

    // Methods
    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private FundingMethod fundingMethod;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private PayoutMethod payoutMethod;

    // Recipient
    @Column(nullable = false)
    private UUID recipientId;

    // Funding details (for bank transfer)
    private String fundingBankName;
    private String fundingAccountNumber;
    private String fundingAccountName;
    private String fundingReference;

    // Timestamps
    @Column(nullable = false, updatable = false)
    private Instant createdAt;

    @Column(nullable = false)
    private Instant updatedAt;

    private Instant completedAt;

    // Failure info
    private String failureReason;

    @PrePersist
    protected void onCreate() {
        createdAt = Instant.now();
        updatedAt = Instant.now();
        if (status == null) {
            status = TransferStatus.CREATED;
        }
    }

    @PreUpdate
    protected void onUpdate() {
        updatedAt = Instant.now();
    }
}

