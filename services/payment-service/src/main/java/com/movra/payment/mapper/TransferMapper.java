package com.movra.payment.mapper;

import com.movra.payment.dto.FundingDetailsResponse;
import com.movra.payment.dto.TransferResponse;
import com.movra.payment.model.FundingMethod;
import com.movra.payment.model.Transfer;
import org.springframework.stereotype.Component;

@Component
public class TransferMapper {

    private final RecipientMapper recipientMapper;

    public TransferMapper(RecipientMapper recipientMapper) {
        this.recipientMapper = recipientMapper;
    }

    public TransferResponse toResponse(Transfer transfer) {
        return TransferResponse.builder()
                .id(transfer.getId().toString())
                .userId(transfer.getUserId().toString())
                .status(transfer.getStatus())
                .sourceCurrency(transfer.getSourceCurrency())
                .sourceAmount(transfer.getSourceAmount())
                .targetCurrency(transfer.getTargetCurrency())
                .targetAmount(transfer.getTargetAmount())
                .exchangeRate(transfer.getExchangeRate())
                .rateLockId(transfer.getRateLockId())
                .rateExpiresAt(transfer.getRateExpiresAt())
                .feeAmount(transfer.getFeeAmount())
                .feeCurrency(transfer.getFeeCurrency())
                .fundingMethod(transfer.getFundingMethod())
                .payoutMethod(transfer.getPayoutMethod())
                .recipientId(transfer.getRecipientId().toString())
                .fundingDetails(buildFundingDetails(transfer))
                .createdAt(transfer.getCreatedAt())
                .updatedAt(transfer.getUpdatedAt())
                .completedAt(transfer.getCompletedAt())
                .failureReason(transfer.getFailureReason())
                .build();
    }

    private FundingDetailsResponse buildFundingDetails(Transfer transfer) {
        if (transfer.getFundingMethod() != FundingMethod.BANK_TRANSFER) {
            return null;
        }
        return FundingDetailsResponse.builder()
                .bankName(transfer.getFundingBankName())
                .accountNumber(transfer.getFundingAccountNumber())
                .accountName(transfer.getFundingAccountName())
                .reference(transfer.getFundingReference())
                .build();
    }
}
