package com.movra.payment.dto;

import com.movra.payment.model.PayoutMethod;
import lombok.Builder;
import lombok.Data;

import java.time.Instant;

@Data
@Builder
public class RecipientResponse {
    private String id;
    private String userId;
    private String nickname;
    private PayoutMethod type;
    private String country;
    private String currency;

    // Bank account details
    private String bankName;
    private String bankCode;
    private String accountNumber;
    private String accountName;

    // Mobile wallet details
    private String walletProvider;
    private String mobileNumber;

    // Cash pickup details
    private String firstName;
    private String lastName;

    private Instant createdAt;
}
