package com.movra.payment.service;

import com.movra.payment.dto.QuoteResponse;
import com.movra.payment.kafka.TransferEventPublisher;
import com.movra.payment.model.*;
import com.movra.payment.repository.TransferRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.time.Instant;
import java.util.UUID;

@Service
@RequiredArgsConstructor
@Slf4j
public class TransferService {

    private final TransferRepository transferRepository;
    private final TransferEventPublisher eventPublisher;

    @Transactional
    public Transfer createTransfer(
            UUID userId,
            String idempotencyKey,
            String sourceCurrency,
            BigDecimal sourceAmount,
            String targetCurrency,
            FundingMethod fundingMethod,
            PayoutMethod payoutMethod,
            UUID recipientId,
            String rateLockId
    ) {
        // Check idempotency
        var existing = transferRepository.findByIdempotencyKey(idempotencyKey);
        if (existing.isPresent()) {
            log.info("Returning existing transfer for idempotency key: {}", idempotencyKey);
            return existing.get();
        }

        // TODO: Get locked rate from Exchange Rate service via gRPC
        // For now, use mock rate
        BigDecimal exchangeRate = getMockRate(sourceCurrency, targetCurrency);
        BigDecimal fee = calculateFee(sourceAmount, fundingMethod);
        BigDecimal netAmount = sourceAmount.subtract(fee);
        BigDecimal targetAmount = netAmount.multiply(exchangeRate).setScale(2, RoundingMode.HALF_UP);

        var transfer = Transfer.builder()
                .userId(userId)
                .idempotencyKey(idempotencyKey)
                .status(TransferStatus.CREATED)
                .sourceCurrency(sourceCurrency)
                .sourceAmount(sourceAmount)
                .targetCurrency(targetCurrency)
                .targetAmount(targetAmount)
                .exchangeRate(exchangeRate)
                .rateLockId(rateLockId)
                .rateExpiresAt(Instant.now().plusSeconds(30))
                .feeAmount(fee)
                .feeCurrency(sourceCurrency)
                .fundingMethod(fundingMethod)
                .payoutMethod(payoutMethod)
                .recipientId(recipientId)
                .build();

        // Generate funding reference for bank transfer
        if (fundingMethod == FundingMethod.BANK_TRANSFER) {
            transfer.setFundingBankName("DBS Bank");
            transfer.setFundingAccountNumber("1234567890");
            transfer.setFundingAccountName("Movra Pte Ltd");
            transfer.setFundingReference("MOV" + System.currentTimeMillis());
        }

        transfer = transferRepository.save(transfer);

        log.info("Created transfer: {} for user: {}", transfer.getId(), userId);

        // Publish event
        eventPublisher.publishTransferInitiated(transfer);

        return transfer;
    }

    public Transfer getTransfer(UUID transferId, UUID userId) {
        return transferRepository.findById(transferId)
                .filter(t -> t.getUserId().equals(userId))
                .orElseThrow(() -> new IllegalArgumentException("Transfer not found"));
    }

    public Page<Transfer> listTransfers(UUID userId, TransferStatus status, Pageable pageable) {
        if (status != null) {
            return transferRepository.findByUserIdAndStatus(userId, status, pageable);
        }
        return transferRepository.findByUserId(userId, pageable);
    }

    @Transactional
    public Transfer confirmTransfer(UUID transferId, UUID userId) {
        var transfer = getTransfer(transferId, userId);

        if (transfer.getStatus() != TransferStatus.CREATED) {
            throw new IllegalStateException("Transfer cannot be confirmed in current status: " + transfer.getStatus());
        }

        // Check if rate lock is still valid
        if (transfer.getRateExpiresAt() != null && Instant.now().isAfter(transfer.getRateExpiresAt())) {
            throw new IllegalStateException("Rate lock has expired. Please create a new transfer.");
        }

        transfer.setStatus(TransferStatus.AWAITING_FUNDS);
        transfer = transferRepository.save(transfer);

        log.info("Transfer confirmed: {}", transferId);

        return transfer;
    }

    @Transactional
    public Transfer cancelTransfer(UUID transferId, UUID userId, String reason) {
        var transfer = getTransfer(transferId, userId);

        if (!transfer.getStatus().canTransitionTo(TransferStatus.CANCELLED)) {
            throw new IllegalStateException("Transfer cannot be cancelled in current status: " + transfer.getStatus());
        }

        transfer.setStatus(TransferStatus.CANCELLED);
        transfer.setFailureReason(reason);
        transfer = transferRepository.save(transfer);

        log.info("Transfer cancelled: {} - reason: {}", transferId, reason);

        return transfer;
    }

    @Transactional
    public Transfer updateStatus(UUID transferId, TransferStatus newStatus, String reason) {
        var transfer = transferRepository.findById(transferId)
                .orElseThrow(() -> new IllegalArgumentException("Transfer not found"));

        if (!transfer.getStatus().canTransitionTo(newStatus)) {
            throw new IllegalStateException(
                    "Invalid status transition: " + transfer.getStatus() + " -> " + newStatus
            );
        }

        transfer.setStatus(newStatus);
        if (reason != null) {
            transfer.setFailureReason(reason);
        }
        if (newStatus == TransferStatus.COMPLETED) {
            transfer.setCompletedAt(Instant.now());
        }

        transfer = transferRepository.save(transfer);

        log.info("Transfer {} status updated: {} -> {}", transferId, transfer.getStatus(), newStatus);

        return transfer;
    }

    public QuoteResponse getQuote(String sourceCurrency, BigDecimal sourceAmount,
                                  String targetCurrency, FundingMethod fundingMethod) {
        // Get rate (mock for now, will integrate with Exchange Rate Service later)
        BigDecimal exchangeRate = getMockRate(sourceCurrency, targetCurrency);
        BigDecimal fee = calculateFee(sourceAmount, fundingMethod);
        BigDecimal netAmount = sourceAmount.subtract(fee);
        BigDecimal targetAmount = netAmount.multiply(exchangeRate).setScale(2, RoundingMode.HALF_UP);

        return QuoteResponse.builder()
                .sourceCurrency(sourceCurrency)
                .sourceAmount(sourceAmount)
                .targetCurrency(targetCurrency)
                .targetAmount(targetAmount)
                .exchangeRate(exchangeRate)
                .fee(fee)
                .totalCost(sourceAmount)
                .validUntil(Instant.now().plusSeconds(30))
                .build();
    }

    private BigDecimal getMockRate(String from, String to) {
        // Mock rates - in production, call Exchange Rate service
        if (from.equals("SGD") && to.equals("PHP")) {
            return new BigDecimal("39.70");
        } else if (from.equals("SGD") && to.equals("INR")) {
            return new BigDecimal("62.50");
        } else if (from.equals("SGD") && to.equals("IDR")) {
            return new BigDecimal("11800");
        } else if (from.equals("USD") && to.equals("PHP")) {
            return new BigDecimal("53.75");
        } else if (from.equals("SGD") && to.equals("USD")) {
            return new BigDecimal("0.74");
        }
        return BigDecimal.ONE;
    }

    private BigDecimal calculateFee(BigDecimal amount, FundingMethod method) {
        // Fee calculation based on funding method
        BigDecimal feePercentage = switch (method) {
            case BANK_TRANSFER -> new BigDecimal("0.005"); // 0.5%
            case CARD -> new BigDecimal("0.025"); // 2.5%
            case WALLET -> new BigDecimal("0.003"); // 0.3%
        };

        BigDecimal fee = amount.multiply(feePercentage).setScale(2, RoundingMode.HALF_UP);
        BigDecimal minFee = new BigDecimal("3.00");

        return fee.max(minFee);
    }
}
