package com.movra.payment.mapper;

import com.movra.payment.dto.CreateRecipientRequest;
import com.movra.payment.dto.RecipientResponse;
import com.movra.payment.model.Recipient;
import org.springframework.stereotype.Component;

import java.util.UUID;

@Component
public class RecipientMapper {

    public RecipientResponse toResponse(Recipient recipient) {
        return RecipientResponse.builder()
                .id(recipient.getId().toString())
                .userId(recipient.getUserId().toString())
                .nickname(recipient.getNickname())
                .type(recipient.getType())
                .country(recipient.getCountry())
                .currency(recipient.getCurrency())
                .bankName(recipient.getBankName())
                .bankCode(recipient.getBankCode())
                .accountNumber(recipient.getAccountNumber())
                .accountName(recipient.getAccountName())
                .walletProvider(recipient.getWalletProvider())
                .mobileNumber(recipient.getMobileNumber())
                .firstName(recipient.getFirstName())
                .lastName(recipient.getLastName())
                .createdAt(recipient.getCreatedAt())
                .build();
    }

    public Recipient toEntity(CreateRecipientRequest request, UUID userId) {
        return Recipient.builder()
                .userId(userId)
                .nickname(request.getNickname())
                .type(request.getType())
                .country(request.getCountry().toUpperCase())
                .currency(request.getCurrency().toUpperCase())
                .bankName(request.getBankName())
                .bankCode(request.getBankCode())
                .accountNumber(request.getAccountNumber())
                .accountName(request.getAccountName())
                .walletProvider(request.getWalletProvider())
                .mobileNumber(request.getMobileNumber())
                .firstName(request.getFirstName())
                .lastName(request.getLastName())
                .build();
    }
}
