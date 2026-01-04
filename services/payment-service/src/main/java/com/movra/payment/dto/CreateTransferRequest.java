package com.movra.payment.dto;

import com.movra.payment.model.FundingMethod;
import jakarta.validation.constraints.*;
import lombok.Data;

import java.math.BigDecimal;

@Data
public class CreateTransferRequest {

    @NotBlank(message = "Idempotency key is required")
    private String idempotencyKey;

    @NotBlank(message = "Source currency is required")
    @Size(min = 3, max = 3, message = "Currency must be 3 characters")
    private String sourceCurrency;

    @NotNull(message = "Source amount is required")
    @DecimalMin(value = "1.00", message = "Minimum amount is 1.00")
    @DecimalMax(value = "50000.00", message = "Maximum amount is 50,000.00")
    private BigDecimal sourceAmount;

    @NotBlank(message = "Target currency is required")
    @Size(min = 3, max = 3, message = "Currency must be 3 characters")
    private String targetCurrency;

    @NotNull(message = "Funding method is required")
    private FundingMethod fundingMethod;

    @NotNull(message = "Recipient ID is required")
    private String recipientId;

    private String rateLockId;
}
