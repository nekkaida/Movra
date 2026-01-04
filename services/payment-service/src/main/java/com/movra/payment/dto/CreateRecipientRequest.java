package com.movra.payment.dto;

import com.movra.payment.model.PayoutMethod;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;
import lombok.Data;

@Data
public class CreateRecipientRequest {

    @NotBlank(message = "Nickname is required")
    @Size(max = 50, message = "Nickname must be 50 characters or less")
    private String nickname;

    @NotNull(message = "Payout method is required")
    private PayoutMethod type;

    @NotBlank(message = "Country is required")
    @Size(min = 2, max = 2, message = "Country must be 2 characters (ISO)")
    private String country;

    @NotBlank(message = "Currency is required")
    @Size(min = 3, max = 3, message = "Currency must be 3 characters")
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
}
