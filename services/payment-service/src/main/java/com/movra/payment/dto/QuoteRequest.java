package com.movra.payment.dto;

import com.movra.payment.model.FundingMethod;
import com.movra.payment.model.PayoutMethod;
import jakarta.validation.constraints.*;
import lombok.Data;

import java.math.BigDecimal;

@Data
public class QuoteRequest {
    @NotBlank(message = "Source currency is required")
    @Size(min = 3, max = 3)
    private String sourceCurrency;

    @NotNull(message = "Source amount is required")
    @DecimalMin(value = "1.00")
    private BigDecimal sourceAmount;

    @NotBlank(message = "Target currency is required")
    @Size(min = 3, max = 3)
    private String targetCurrency;

    @NotNull(message = "Funding method is required")
    private FundingMethod fundingMethod;

    private PayoutMethod payoutMethod;
}
